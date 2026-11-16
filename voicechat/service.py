"""Local Telegram voice transport; the Go application owns the music queue."""
from __future__ import annotations

import asyncio
import contextlib
import hmac
import logging
import math
import os
import signal
from pathlib import Path
from urllib.parse import urlsplit

from aiohttp import web


class AssistantMissing(Exception):
    pass

class AssistantBanned(Exception):
    pass

class JoinPending(Exception):
    pass

class JoinDeclined(Exception):
    pass


async def ensure_membership(user, chat: int, invite: str = "") -> None:
    async def check():
        member = await user.get_chat_member(chat, "me")
        status = str(member.status).lower()
        if status.endswith(("banned", "kicked")):
            raise AssistantBanned()
        if status.endswith("left") or getattr(member,"is_member",None) is False:
            raise AssistantMissing()

    missing = {"PeerIdInvalid", "ChannelPrivate", "ChannelInvalid", "UserNotParticipant", "ChatIdInvalid", "AssistantMissing", "ChatAdminRequired"}
    try:
        await check()
        return
    except Exception as error:
        if type(error).__name__ not in missing:
            raise
    # String sessions may know the membership but lack the peer access hash.
    if not invite:
        async for dialog in user.get_dialogs():
            if dialog.chat.id == chat:
                try:
                    await check()
                    return
                except Exception as error:
                    if type(error).__name__ not in missing:
                        raise
                    break
    if not invite:
        raise AssistantMissing()
    if not isinstance(invite, str) or not invite.startswith("https://t.me/"):
        raise ValueError("invalid invite")
    try:
        joined = await user.join_chat(invite)
        result_type = type(joined).__name__
        if result_type in {"ChatJoinResultRequestSent", "ChatJoinResultGuardBotApprovalRequired"}:
            raise JoinPending()
        if result_type == "ChatJoinResultDeclined":
            raise JoinDeclined()
        joined_chat = getattr(joined, "chat", joined)
        if getattr(joined_chat, "id", None) != chat:
            raise ValueError("invite target mismatch")
    except Exception as error:
        if type(error).__name__ != "UserAlreadyParticipant":
            raise
    for attempt in range(4):
        try:
            await check()
            return
        except Exception as error:
            if type(error).__name__ not in missing or attempt==3:
                raise
            await asyncio.sleep(0.25*(2**attempt))


def error_code(error: Exception) -> str:
    name = type(error).__name__
    if name in {"NoActiveGroupCall", "NotInCallError", "GroupCallNotFound", "GroupcallInvalid", "GroupCallInvalid"}:
        return "call_missing"
    if name in {"PeerIdInvalid", "ChannelPrivate", "ChannelInvalid", "UserNotParticipant", "ChatIdInvalid", "AssistantMissing"}:
        return "assistant_missing"
    if name in {"ChatAdminRequired", "Forbidden", "GroupcallForbidden", "GroupCallForbidden"}:
        return "permission"
    if name in {"AuthKeyUnregistered", "SessionRevoked", "SessionExpired", "UserDeactivated"}:
        return "session"
    if name == "FloodWait":
        return "flood"
    if name in {"InviteRequestSent", "JoinPending"}:
        return "join_pending"
    if name == "JoinDeclined":
        return "permission"
    if name == "AssistantBanned":
        return "assistant_banned"
    if isinstance(error, FileNotFoundError):
        return "media_missing"
    return "stream_failed"


def valid_source(value: str, root: Path) -> str:
    if not isinstance(value, str) or not value or len(value) > 8192:
        raise ValueError("invalid source")
    url = urlsplit(value)
    if url.scheme in {"http", "https"} and url.hostname:
        return value
    path = Path(value).resolve()
    if not path.is_relative_to(root) or not path.is_file():
        raise FileNotFoundError()
    return str(path)


class Transport:
    def __init__(self, calls, media, call_config, root: Path, prepare=None):
        self.calls, self.media, self.call_config = calls, media, call_config
        self.root = root.resolve()
        self.sources: dict[int, str] = {}
        self.locks: dict[int, asyncio.Lock] = {}
        self.prepare = prepare

    async def execute(self, data: dict) -> None:
        chat = data.get("chat_id")
        if type(chat) is not int or chat >= 0:
            raise ValueError("group required")
        async with self.locks.setdefault(chat, asyncio.Lock()):
            action = data.get("action")
            if action == "prepare":
                if self.prepare is not None:
                    await self.prepare(chat, data.get("invite", ""))
            elif action == "play":
                source = valid_source(data.get("source"), self.root)
                if self.prepare is not None:
                    await self.prepare(chat, "")
                await self.calls.play(chat, self.media(source, 0), self.call_config())
                self.sources[chat] = source
            elif action == "stop":
                if chat in self.sources:
                    try:
                        await self.calls.leave_call(chat)
                    except Exception as error:
                        if error_code(error) != "call_missing":
                            raise
                    self.sources.pop(chat, None)
            elif action in {"pause", "resume"}:
                if chat not in self.sources:
                    raise ValueError("no stream")
                await getattr(self.calls, action)(chat)
            elif action == "seek":
                position = data.get("position", 0)
                if type(position) not in {int, float} or not math.isfinite(position) or position < 0:
                    raise ValueError("invalid position")
                if chat not in self.sources:
                    raise ValueError("no stream")
                await self.calls.play(chat, self.media(self.sources[chat], position), self.call_config())
                if data.get("paused") is True:
                    await self.calls.pause(chat)
            else:
                raise ValueError("unknown action")


def create_app(transport: Transport, token: str, assistant_id: int = 0) -> web.Application:
    @web.middleware
    async def authenticate(request, handler):
        if not hmac.compare_digest(request.headers.get("Authorization", ""), "Bearer " + token):
            return web.json_response({"code": "unauthorized"}, status=401)
        return await handler(request)

    async def command(request):
        data = None
        try:
            data = await request.json()
            if not isinstance(data, dict):
                raise ValueError("object required")
            await asyncio.wait_for(transport.execute(data), timeout=45)
            return web.json_response({"ok": True})
        except (ValueError, TypeError):
            return web.json_response({"code": "invalid_request"}, status=400)
        except Exception as error:
            # Do not log raw RPC errors, media URLs, credentials, or session strings.
            logging.warning("Voice command failed: %s (action %s, chat %s)", type(error).__name__, data.get("action") if isinstance(data, dict) else "unknown", data.get("chat_id") if isinstance(data, dict) else "unknown")
            return web.json_response({"code": error_code(error)}, status=502)

    async def health(_request):
        return web.json_response({"ready": True,"assistant_id":assistant_id})

    app = web.Application(middlewares=[authenticate], client_max_size=16384)
    app.router.add_get("/health", health)
    app.router.add_post("/command", command)
    return app


async def run() -> None:
    # Delayed imports let transport tests run without opening a Telegram session.
    from pyrogram import Client
    from pytgcalls import PyTgCalls
    from pytgcalls.types import AudioQuality, GroupCallConfig, MediaStream

    token = os.environ["VOICE_BRIDGE_TOKEN"]
    session = os.environ.get("STRING_SESSION", "").strip()
    if len(token) < 32 or not session:
        raise ValueError("missing assistant configuration")
    user = Client("voice-assistant", api_id=int(os.environ["API_ID"]),
                  api_hash=os.environ["API_HASH"], session_string=session,
                  in_memory=True)
    # Never prompt for a phone number, login code or password in a deployed bot.
    if not await user.connect():
        raise ValueError("assistant session is not authorized")
    me = await user.get_me()
    if me.is_bot:
        await user.disconnect()
        raise ValueError("STRING_SESSION must belong to a user account")
    user.me = me
    await user.initialize()
    calls = PyTgCalls(user)
    await calls.start()
    # String sessions do not carry the peer cache/access hashes. Load the
    # assistant's groups so private group IDs can be resolved on the first play.
    async for _dialog in user.get_dialogs():
        pass

    def media(source, position):
        return MediaStream(source, AudioQuality.HIGH,
                           video_flags=MediaStream.Flags.IGNORE,
                           ffmpeg_parameters=f"-ss {float(position):.3f}" if position else None)

    transport = Transport(calls, media, lambda: GroupCallConfig(auto_start=True), Path.cwd(),
                          lambda chat, invite: ensure_membership(user, chat, invite))
    runner = web.AppRunner(create_app(transport, token,me.id), access_log=None)
    await runner.setup()
    await web.TCPSite(runner, "127.0.0.1", 9091).start()
    logging.warning("Voice assistant ready: @%s (ID %d)", me.username or "no_username", me.id)
    stopped = asyncio.Event()
    loop = asyncio.get_running_loop()
    for sig in (signal.SIGINT, signal.SIGTERM):
        with contextlib.suppress(NotImplementedError):
            loop.add_signal_handler(sig, stopped.set)
    try:
        await stopped.wait()
    finally:
        await runner.cleanup()
        for chat in list(transport.sources):
            with contextlib.suppress(Exception):
                await calls.leave_call(chat)
        await user.stop()


if __name__ == "__main__":
    logging.basicConfig(level=logging.WARNING)
    try:
        asyncio.run(run())
    except KeyboardInterrupt:
        pass
    except Exception as failure:
        logging.error("Voice assistant startup failed: %s", type(failure).__name__)
        raise SystemExit(1) from None
