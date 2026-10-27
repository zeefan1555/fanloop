'use strict';

const fs = require('fs');
const path = require('path');
const crypto = require('crypto');
const {isDeepStrictEqual} = require('node:util');
const layout = require('./layout');
const {commands} = require('./commands.json');
const digestPattern = /^sha256:[0-9a-f]{64}$/;
const idPattern = /^[0-9a-f]{32}$/;
const versionPattern = /^[0-9A-Za-z][0-9A-Za-z._+-]*$/;
const skillNames = ['codex', 'agent', 'trae', 'claude'];
const skillEnv = ['FANLOOP_CODEX_SKILLS_ROOT','FANLOOP_AGENT_SKILLS_ROOT','FANLOOP_TRAE_SKILLS_ROOT','FANLOOP_CLAUDE_SKILLS_ROOT'];

function absolute(value) {
  return typeof value === 'string' && value !== '/' && path.isAbsolute(value) && path.normalize(value) === value && !/[\r\n\0]/.test(value);
}
function exists(value) {
  try { fs.lstatSync(value); return true; } catch (err) { if (err.code === 'ENOENT') return false; throw err; }
}
function timestamp(value) {
  if (typeof value !== 'string') return false;
  const fields = /^(\d{4})-(\d\d)-(\d\d)T(\d\d):(\d\d):(\d\d)(?:\.\d+)?(?:Z|[+-]\d\d:\d\d)$/.exec(value);
  if (!fields || !Number.isFinite(Date.parse(value))) return false;
  const [year, month, day, hour, minute, second] = fields.slice(1).map(Number);
  const leap = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
  const days = [31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  return month >= 1 && month <= 12 && day >= 1 && day <= days[month - 1] && hour < 24 && minute < 60 && second < 60;
}
function decode(content) {
  const value = JSON.parse(content);
  const fields = ['schema_version','transaction_id','phase','configuration_mode','source_release_path','controller_release_path','release_manifest_sha256','binary_sha256','previous_configuration_pin','configuration_pin','configuration_manifest_sha256','prepared_at','committed_at','operation','configuration_source_path','previous_binding_transaction_id','protected_files_sha256','release_resources_sha256'];
  if (!value || typeof value !== 'object' || Array.isArray(value) || Object.keys(value).some(key => !fields.includes(key))) throw new Error('invalid controller journal fields');
  const optional = ['configuration_manifest_sha256','committed_at','previous_binding_transaction_id','protected_files_sha256'];
  for (const key of fields.filter(key => key !== 'schema_version')) {
    if (!(key in value) && optional.includes(key)) continue;
    if (typeof value[key] !== 'string') throw new Error('controller journal field must be a string: '+key);
  }
  if (value.schema_version !== 1 || !idPattern.test(value.transaction_id) || !['prepared','committed'].includes(value.phase) || !['snapshot','author_live','author_frozen'].includes(value.configuration_mode) || !['initialize','protect'].includes(value.operation)) throw new Error('invalid controller journal identity');
  for (const key of ['source_release_path','configuration_source_path','configuration_pin']) if (!absolute(value[key])) throw new Error('invalid controller journal path');
  if (typeof value.previous_configuration_pin !== 'string' || (value.previous_configuration_pin && !absolute(value.previous_configuration_pin))) throw new Error('invalid previous pin');
  const parts = value.controller_release_path.split('/');
  if (parts.length !== 2 || parts[0] !== 'releases' || !versionPattern.test(parts[1])) throw new Error('invalid relative controller release');
  for (const key of ['release_manifest_sha256','binary_sha256','release_resources_sha256']) if (!digestPattern.test(value[key] || '')) throw new Error('invalid controller digest');
  for (const key of ['configuration_manifest_sha256','protected_files_sha256']) if (key in value && !digestPattern.test(value[key])) throw new Error('invalid optional digest');
  if (!timestamp(value.prepared_at) || (value.phase === 'committed' ? !timestamp(value.committed_at) : 'committed_at' in value)) throw new Error('invalid controller journal time');
  if (value.configuration_mode === 'author_live') {
    if ('configuration_manifest_sha256' in value || value.configuration_pin !== value.configuration_source_path) throw new Error('author/live must retain editable source');
  } else if (!value.configuration_manifest_sha256) throw new Error('immutable configuration needs identity');
  if (value.operation === 'initialize') {
    if (value.previous_configuration_pin || 'protected_files_sha256' in value || 'previous_binding_transaction_id' in value) throw new Error('initialize cannot carry protect history');
  } else if (!value.protected_files_sha256) throw new Error('protect needs business file identity');
  if ('previous_binding_transaction_id' in value && !idPattern.test(value.previous_binding_transaction_id)) throw new Error('invalid previous controller transaction');
  return value;
}
function readBinding(root) {
  const file = path.join(root,'.fanloop','controller-binding.json');
  if (!exists(file)) return null;
  layout.realPath(file,false);
  if ((fs.lstatSync(file).mode & 0o777) !== 0o600) throw new Error('journal must be 0600');
  const value = decode(fs.readFileSync(file,'utf8'));
  if (value.configuration_mode !== 'author_live') {
    const expected = value.configuration_mode === 'author_frozen'
      ? path.join(root,'bound-release-home','config','author',value.configuration_manifest_sha256.slice(7))
      : path.join(root,'bound-release-home','config','snapshots',path.basename(value.configuration_source_path));
    if (value.configuration_pin !== expected) throw new Error('journal configuration target escapes its fixed controller home');
  }
  if (value.previous_binding_transaction_id) {
    const archive = path.join(root,'.fanloop','audit','controller-bindings',value.previous_binding_transaction_id+'.json');
    layout.realPath(archive,false);
    const previous = decode(fs.readFileSync(archive,'utf8'));
    if (previous.transaction_id !== value.previous_binding_transaction_id || previous.phase !== 'committed' || previous.configuration_mode !== 'author_live' || previous.configuration_pin !== value.previous_configuration_pin || previous.controller_release_path !== value.controller_release_path || previous.release_manifest_sha256 !== value.release_manifest_sha256 || previous.release_resources_sha256 !== value.release_resources_sha256) throw new Error('previous committed journal does not match');
  }
  return value;
}
function readPin(root) {
  const file = path.join(root,'.fanloop','configuration-snapshot');
  if (!exists(file)) return '';
  layout.realPath(file,false);
  const pin = fs.readFileSync(file,'utf8').replace(/\n$/, '');
  if (!absolute(pin)) throw new Error('configuration pin must contain one clean absolute path');
  return pin;
}
function boundCurrent(home) {
  layout.realPath(home,true);
  const current = path.join(home,'current');
  const managed = layout.assertCurrentReplaceable(home);
  const binding = managed ? path.join(current,'release') : current;
  const raw = fs.readlinkSync(binding);
  const target = path.resolve(path.dirname(binding),raw);
  if (path.dirname(target) !== path.join(home,'releases') || !versionPattern.test(path.basename(target))) throw new Error('controller current escapes releases');
  if (raw !== target && raw !== path.relative(path.dirname(binding),target)) throw new Error('invalid controller current primary link');
  layout.realPath(target,true);
  if (managed) for (const [name,expected] of Object.entries({bin:'release/bin',entrypoints:'release/entrypoints','release.json':'release/release.json'})) if (fs.readlinkSync(path.join(current,name)) !== expected) throw new Error('invalid current alias');
  return target;
}
function configurationManifest(raw) {
  const object = value => {
    if (value === null) return {};
    if (!value || typeof value !== 'object' || Array.isArray(value)) throw new Error('invalid configuration manifest object');
    return value;
  };
  const scalar = (value, type, empty) => {
    if (value === undefined || value === null) return empty;
    if (typeof value !== type || type === 'number' && !Number.isSafeInteger(value)) throw new Error('invalid configuration manifest field type');
    return value;
  };
  const string = value => scalar(value, 'string', '');
  const list = (value, item) => {
    if (value === undefined || value === null) return null;
    if (!Array.isArray(value)) throw new Error('invalid configuration manifest list');
    return value.map(item);
  };
  raw = object(raw);
  return {
    schema_version: scalar(raw.schema_version, 'number', 0),
    origin: string(raw.origin),
    repository: string(raw.repository),
    ref: string(raw.ref),
    commit_sha: string(raw.commit_sha),
    release_set_digest: string(raw.release_set_digest),
    releases: list(raw.releases, value => {
      const release = object(value);
      return {key:string(release.key), version:string(release.version), workflow_id:string(release.workflow_id)};
    }),
    workflow_ids: list(raw.workflow_ids, string),
    files: list(raw.files, value => {
      const file = object(value);
      return {path:string(file.path), sha256:string(file.sha256), executable:scalar(file.executable, 'boolean', false)};
    }),
    configuration_manifest_sha256: string(raw.configuration_manifest_sha256),
  };
}
function configurationIdentity(source) {
  layout.realPath(source,true);
  const root = path.join(source,'fanloop-workflow');
  layout.realPath(root,true);
  const files = [];
  const workflowIDs = [];
  const byteOrder = (a,b) => Buffer.compare(Buffer.from(a),Buffer.from(b));
  const walk = dir => {
    for (const name of fs.readdirSync(dir).sort(byteOrder)) {
      const full = path.join(dir,name);
      const info = fs.lstatSync(full);
      if (info.isDirectory()) {
        if (dir === root) workflowIDs.push(name);
        walk(full);
      }
      else if (info.isFile()) files.push({path:path.relative(source,full).split(path.sep).join('/'),sha256:layout.fileDigest(full).slice(7),executable:(info.mode & 0o111) !== 0});
      else throw new Error('unsupported configuration resource');
    }
  };
  walk(root);
  const json = JSON.stringify(files).replace(/[<>&\u2028\u2029]/g,c => '\\u'+c.charCodeAt(0).toString(16).padStart(4,'0'));
  const digest = 'sha256:'+crypto.createHash('sha256').update(json).digest('hex');
  const managed = path.basename(path.dirname(source)) === 'snapshots' && path.basename(path.dirname(path.dirname(source))) === 'config';
  const manifestPath = path.join(source,'configuration.json');
  if (managed || exists(manifestPath)) {
    layout.realPath(manifestPath,false);
    const manifest = configurationManifest(JSON.parse(fs.readFileSync(manifestPath,'utf8')));
    const actual = {
      schema_version:1, origin:'', repository:manifest.repository, ref:'refs/heads/main',
      commit_sha:manifest.commit_sha, release_set_digest:'', releases:null,
      workflow_ids:workflowIDs, files, configuration_manifest_sha256:digest,
    };
    if (manifest.origin === 'fornax') {
      const releases = manifest.releases && manifest.releases.length ? [...manifest.releases].sort((a,b) => byteOrder(a.key,b.key)) : null;
      const present = new Set(workflowIDs);
      if ((releases || []).length !== present.size || (releases || []).some(r => !r.key || !r.version || !r.workflow_id || !present.has(r.workflow_id))) throw new Error('Fornax releases differ from workflow packages');
      const lines = (releases || []).map(r => r.key+'@'+r.version).sort(byteOrder);
      const identity = crypto.createHash('sha256').update(lines.join('\n')+'\n').digest('hex');
      if (path.basename(source) !== identity || manifest.release_set_digest !== identity) throw new Error('Fornax configuration identity differs');
      Object.assign(actual, {origin:'fornax',repository:'',ref:'',commit_sha:'',release_set_digest:identity,releases});
    } else if (!/^[0-9a-f]{40}$/.test(manifest.commit_sha || '') || manifest.commit_sha !== path.basename(source)) throw new Error('managed configuration commit differs');
    if (!isDeepStrictEqual(manifest,actual)) throw new Error('managed configuration identity differs');
  }
  return digest;
}
function invocation(args) {
  const allFlags = Object.assign({},...Object.values(commands).map(c => c.flags));
  const input = [...args];
  const words = [];
  const flags = [];
  let root = '';
  let help = false;
  let dryRun = false;
  for (let i=0;i<input.length;i++) {
    let argument=input[i];
    if (argument === '--') break;
    if (!argument.startsWith('-') || argument === '-') { words.push(argument); continue; }
    if (!argument.startsWith('--') && argument.length > 2 && argument[2] !== '=') {
      const shorthand=argument.slice(0,2);
      if (!(shorthand in allFlags)) return null;
      input.splice(i,1,shorthand,allFlags[shorthand] ? '-'+argument.slice(2) : argument.slice(2));
      argument=shorthand;
    }
    const equal=argument.indexOf('=');
    const name=equal < 0 ? argument : argument.slice(0,equal);
    if (!(name in allFlags)) return null;
    const boolean=allFlags[name];
    const value=equal < 0 ? boolean ? 'true' : input[++i] : argument.slice(equal+1);
    if (value === undefined) return null;
    let parsed=value;
    if (boolean) {
      if (['1','t','T','TRUE','true','True'].includes(value)) parsed=true;
      else if (['0','f','F','FALSE','false','False'].includes(value)) parsed=false;
      else return null;
    }
    flags.push(name);
    if (name === '--root') root=value;
    if (name === '--help' || name === '-h') help=parsed;
    if (name === '--dry-run') dryRun=parsed;
  }
  let key=words.join(' ');
  while (!(key in commands) && words.length) {
    words.pop();
    key=words.join(' ');
  }
  const spec=commands[key];
  if (!spec || spec.scope === 'none' || help || !path.isAbsolute(root) || flags.some(flag => !(flag in spec.flags))) return null;
  return {root, rebind:key === 'config rebind', command:key.replace(/ /g,'.'), dryRun};
}
function select(args, env=process.env) {
  const request=invocation(args);
  if (!request) return null;
  const {root}=request;
  const journal=readBinding(root);
  if (journal && journal.phase !== 'committed') throw new Error('prepared controller binding is blocked; initialize is never resumed');
  const home=path.join(root,'bound-release-home');
  if (!exists(home)) {
    if (journal) throw new Error('committed controller home is missing');
    return null;
  }
  const physical=boundCurrent(home);
  const manifest=layout.validateReleaseTree(physical,path.basename(physical),true);
  if (journal && (physical !== path.join(home,journal.controller_release_path) || manifest.cli.binary_sha256 !== journal.binary_sha256 || layout.fileDigest(path.join(physical,'release.json')) !== journal.release_manifest_sha256 || layout.controllerResourcesDigest(physical) !== journal.release_resources_sha256)) throw new Error('committed controller identity differs');
  for (const name of skillNames) {
    const skillRoot=path.join(home,'skill-roots',name);
    layout.realPath(skillRoot,true);
    for (const skill of layout.EXPOSED_SKILLS) if (fs.readlinkSync(path.join(skillRoot,skill.name)) !== path.join(home,'current',skill.path)) throw new Error('controller Skill link differs');
  }
  const pin=readPin(root);
  if (!pin && journal) throw new Error('bound configuration pin is missing');
  if (pin) {
    const digest=configurationIdentity(pin);
    if (journal && pin === journal.configuration_pin && journal.configuration_mode !== 'author_live' && digest !== journal.configuration_manifest_sha256) throw new Error('protected configuration identity differs');
  }
  const childEnv={...env,FANLOOP_DATA_HOME:home,FANLOOP_NO_AUTO_CONFIG:'1',FANLOOP_NO_AUTO_UPDATE:'1'};
  if (pin) childEnv.FANLOOP_CONFIG_SOURCE=pin; else delete childEnv.FANLOOP_CONFIG_SOURCE;
  for (let i=0;i<skillNames.length;i++) childEnv[skillEnv[i]]=path.join(home,'skill-roots',skillNames[i]);
  if (request.rebind) {
    const caller=layout.dataHome(env);
    layout.realPath(path.join(caller,'config','snapshots'),true);
    childEnv.FANLOOP_DATA_HOME=caller;
  }
  return {binary:path.join(physical,'bin','fanloop'),env:childEnv,authorLive:journal && journal.configuration_mode === 'author_live' && pin === journal.configuration_pin};
}

module.exports={select,invocation,decode,boundCurrent,configurationIdentity};
