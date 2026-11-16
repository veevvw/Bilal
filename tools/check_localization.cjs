const fs=require('fs'),path=require('path'),assert=require('assert/strict'),{spawnSync}=require('child_process');
const base=path.resolve('internal/webapp/static');
const files=fs.readdirSync(path.join(base,'js')).filter(x=>x.endsWith('.js')).map(x=>path.join(base,'js',x)).concat(path.join(base,'app.js'));
for(const file of files){const result=spawnSync(process.execPath,['--check',file],{encoding:'utf8'});assert.equal(result.status,0,result.stderr);}
const catalog=JSON.parse(fs.readFileSync('internal/i18n/ar.json','utf8'));
const fmt=/%(?:\[[0-9]+\])?[-+#0 ]*[0-9]*(?:\.[0-9]+)?[a-zA-Z%]/g;
for(const [source,translated]of Object.entries(catalog))assert.deepEqual(source.match(fmt)||[],translated.match(fmt)||[],`Format arguments: ${source}`);
for(const name of ['home.html','privacy.html','room.html']){
 const html=fs.readFileSync(path.join(base,name),'utf8');assert.match(html,/<html lang="ar" dir="rtl">/);assert.match(html,/id="ui-language"/);assert.ok(!/\?{3,}/.test(html));
}
assert.equal(catalog['Home'],'الرئيسية');
console.log(`PASS JavaScript syntax, format arguments, and Arabic defaults (${Object.keys(catalog).length} bot templates)`);
