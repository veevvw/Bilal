import asyncio
import tempfile
import unittest
from pathlib import Path

from aiohttp.test_utils import TestClient, TestServer
from service import Transport, create_app, ensure_membership, AssistantMissing, error_code, JoinPending
from types import SimpleNamespace


class FakeCalls:
    def __init__(self):
        self.actions = []

    async def play(self, chat, media, config):
        self.actions.append(("play", chat, media))

    async def pause(self, chat):
        self.actions.append(("pause", chat))

    async def resume(self, chat):
        self.actions.append(("resume", chat))

    async def leave_call(self, chat):
        self.actions.append(("stop", chat))


class ServiceTests(unittest.IsolatedAsyncioTestCase):
    async def asyncSetUp(self):
        self.directory = tempfile.TemporaryDirectory()
        self.root = Path(self.directory.name)
        self.source = self.root / "song.mp3"
        self.source.write_bytes(b"test audio")
        self.calls = FakeCalls()
        self.transport = Transport(self.calls, lambda source, pos: (source, pos), lambda: None, self.root)

    async def asyncTearDown(self):
        self.directory.cleanup()

    async def test_play_pause_seek_resume_stop(self):
        await self.transport.execute({"action": "play", "chat_id": -100, "source": str(self.source)})
        await self.transport.execute({"action": "pause", "chat_id": -100})
        await self.transport.execute({"action": "seek", "chat_id": -100, "position": 45, "paused": True})
        await self.transport.execute({"action": "resume", "chat_id": -100})
        await self.transport.execute({"action": "stop", "chat_id": -100})
        await self.transport.execute({"action": "stop", "chat_id": -100})
        self.assertEqual([a[0] for a in self.calls.actions], ["play", "pause", "play", "pause", "resume", "stop"])
        self.assertEqual(self.calls.actions[2][2], (str(self.source), 45))
        self.assertEqual(self.transport.sources, {})

    async def test_invalid_group_source_and_seek(self):
        for data in [
            {"action": "play", "chat_id": 123, "source": str(self.source)},
            {"action": "play", "chat_id": True, "source": str(self.source)},
            {"action": "seek", "chat_id": -100, "position": float("nan")},
        ]:
            with self.assertRaises(ValueError):
                await self.transport.execute(data)
        with self.assertRaises(FileNotFoundError):
            await self.transport.execute({"action": "play", "chat_id": -100, "source": "../private-session"})
        self.assertEqual(self.calls.actions, [])

    async def test_authenticated_local_api(self):
        client = TestClient(TestServer(create_app(self.transport, "test-token")))
        await client.start_server()
        try:
            self.assertEqual((await client.get("/health")).status, 401)
            headers = {"Authorization": "Bearer test-token"}
            self.assertTrue((await (await client.get("/health", headers=headers)).json())["ready"])
            response = await client.post("/command", headers=headers, json={"action":"play", "chat_id":-100, "source":str(self.source)})
            self.assertEqual(response.status, 200)
            self.assertEqual(len(self.calls.actions), 1)
        finally:
            await client.close()

    async def test_commands_are_serialized_per_group(self):
        original = self.calls.play
        started, release = asyncio.Event(), asyncio.Event()
        async def delayed(chat, media, config):
            started.set()
            await release.wait()
            await original(chat, media, config)
        self.calls.play = delayed
        play = asyncio.create_task(self.transport.execute({"action":"play", "chat_id":-100, "source":str(self.source)}))
        await started.wait()
        stop = asyncio.create_task(self.transport.execute({"action":"stop", "chat_id":-100}))
        await asyncio.sleep(0)
        self.assertFalse(stop.done())
        release.set()
        await asyncio.gather(play, stop)
        self.assertEqual([a[0] for a in self.calls.actions], ["play", "stop"])

    async def test_stop_after_group_call_was_closed(self):
        class NotInCallError(Exception):
            pass
        async def already_closed(_chat):
            raise NotInCallError()
        self.transport.sources[-100] = str(self.source)
        self.calls.leave_call = already_closed
        await self.transport.execute({"action":"stop", "chat_id":-100})
        self.assertEqual(self.transport.sources, {})

    async def test_prepare_runs_before_audio(self):
        actions = []
        async def prepare(chat, invite):
            actions.append((chat, invite))
        self.transport.prepare = prepare
        await self.transport.execute({"action":"prepare","chat_id":-123,"invite":"https://t.me/+temporary"})
        await self.transport.execute({"action":"play","chat_id":-123,"source":str(self.source)})
        self.assertEqual(actions,[(-123,"https://t.me/+temporary"),(-123,"")])

    async def test_membership_refresh_and_private_invite(self):
        class FakeUser:
            def __init__(self):
                self.member = False
                self.invites = []
            async def get_chat_member(self, chat, who):
                if not self.member:
                    raise AssistantMissing()
                return SimpleNamespace(status="member")
            async def get_dialogs(self):
                if False:
                    yield None
            async def join_chat(self, invite):
                self.invites.append(invite)
                self.member=True
                return SimpleNamespace(id=-123)
        user=FakeUser()
        with self.assertRaises(AssistantMissing):
            await ensure_membership(user,-123)
        await ensure_membership(user,-123,"https://t.me/+temporary")
        await ensure_membership(user,-123)
        self.assertEqual(user.invites,["https://t.me/+temporary"])

    async def test_wrong_invite_target_never_plays(self):
        class FakeUser:
            async def get_chat_member(self,*args):
                raise AssistantMissing()
            async def get_dialogs(self):
                if False:
                    yield None
            async def join_chat(self,invite):
                return SimpleNamespace(id=-999)
        with self.assertRaises(ValueError):
            await ensure_membership(FakeUser(),-123,"https://t.me/+wrong")

    async def test_invite_joins_without_a_slow_dialog_scan(self):
        class FakeUser:
            member=False
            async def get_chat_member(self,*args):
                if not self.member:
                    raise AssistantMissing()
                return SimpleNamespace(status="member")
            async def get_dialogs(self):
                raise AssertionError("invite should be used immediately")
                yield None
            async def join_chat(self,invite):
                self.member=True
                return SimpleNamespace(id=-123)
        await ensure_membership(FakeUser(),-123,"https://t.me/+temporary")

    async def test_health_exposes_identity_only_after_authentication(self):
        client=TestClient(TestServer(create_app(self.transport,"test-token",77)))
        await client.start_server()
        try:
            self.assertEqual((await client.get("/health")).status,401)
            response=await client.get("/health",headers={"Authorization":"Bearer test-token"})
            self.assertEqual((await response.json())["assistant_id"],77)
        finally:
            await client.close()

    async def test_channel_invalid_requires_invite_instead_of_stream_failure(self):
        class ChannelInvalid(Exception):
            pass
        class FakeUser:
            member=False
            async def get_chat_member(self,*args):
                if not self.member:
                    raise ChannelInvalid()
                return SimpleNamespace(status="member")
            async def get_dialogs(self):
                if False:
                    yield None
            async def join_chat(self,invite):
                self.member=True
                return SimpleNamespace(id=-123)
        self.assertEqual(error_code(ChannelInvalid()),"assistant_missing")
        user=FakeUser()
        with self.assertRaises(AssistantMissing):
            await ensure_membership(user,-123)
        await ensure_membership(user,-123,"https://t.me/+temporary")

    async def test_kurigram_wrapped_chat_join_result(self):
        class ChatJoinResultSuccess:
            def __init__(self):
                self.chat=SimpleNamespace(id=-123)
        class FakeUser:
            member=False
            async def get_chat_member(self,*args):
                if not self.member:
                    raise AssistantMissing()
                return SimpleNamespace(status="member")
            async def join_chat(self,invite):
                self.member=True
                return ChatJoinResultSuccess()
        await ensure_membership(FakeUser(),-123,"https://t.me/+temporary")

    async def test_join_request_result_does_not_start_stream(self):
        class ChatJoinResultRequestSent:
            pass
        class FakeUser:
            async def get_chat_member(self,*args):
                raise AssistantMissing()
            async def join_chat(self,invite):
                return ChatJoinResultRequestSent()
        with self.assertRaises(JoinPending):
            await ensure_membership(FakeUser(),-123,"https://t.me/+temporary")


if __name__ == "__main__":
    unittest.main()
