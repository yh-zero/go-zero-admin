import test from 'node:test';
import assert from 'node:assert/strict';
import {createHash} from 'node:crypto';
import {mkdtemp,writeFile,readFile,realpath,rm} from 'node:fs/promises';
import {resolve,dirname,basename,join} from 'node:path';
import {execFile} from 'node:child_process';
import {promisify} from 'node:util';
import {pathToFileURL} from 'node:url';
import {createApiClient} from './eval-kb-quality.mjs';
const module=await import('./eval-kb-cohort.mjs').catch(error=>{if(error.code==='ERR_MODULE_NOT_FOUND')return {};throw error;});
const feature=name=>{assert.equal(typeof module[name],'function',`缺少完整题集能力 ${name}`);return module[name];};
const hash=value=>createHash('sha256').update(value).digest('hex');
const text='事实甲。',source={key:'a',kb:'project',title:'项目资料',source:'ai_document/ai-rag/a.md',sha256:hash(text)};
const item=id=>({id,split:'development',category:'direct',kb:'project',query:`${id}的事实是什么？`,rationale:'fixture',expected_documents:['a'],evidence:[{document:'a',groups:[['事实甲']]}],answer:{kind:'answerable',required_groups:[['事实甲']],forbidden_phrases:[]}});
const doc={id:11,kb_id:7,title:source.title,version:1,status:'ready',embedding_model:'fixture'};
const hit={chunk_id:101,kb_id:7,doc_id:11,doc_title:source.title,snippet:text,score:.9};
const analysis={version:1,type:'kb_analysis',status:'completed',analysis:'事实甲 [1]',sources:[{...hit,index:1}],usage:{scope:'chat',prompt_tokens:10,completion_tokens:5,total_tokens:15,complete:true}};
const context=()=>({binary_sha:'a'.repeat(64),model:{provider:'fixture',name:'fixture'},embedding:{provider:'fixture',model:'fixture',dimension:3},config:{max_context_tokens:32000,analysis_answer_token_cap:2048,recall_top_k:20,return_top_k:5}});
function fixture({failAt=-1,unknownAt=-1,logFailAt=-1,drift=false}={}){
 const cases=['one','two','three'].map(item),logs=new Map(),events=[],requests=[],calls=[];let next=51,lists=0;
 const envelope=result=>new Response(JSON.stringify({code:200,result}));
 const fetchImpl=async(url,options)=>{
  const u=new URL(url),path=u.pathname.replace('/v1/sys/ai','');calls.push({path,method:options.method});
  if(path==='/kbs/7/docs'){lists++;return envelope({total:1,list:[{...doc,version:drift&&lists>=8?2:1}]});}
  if(path==='/kbs/7/docs/11')return envelope({doc,content:text});
  if(path.endsWith('/search'))return envelope({list:[hit],elapsed_ms:2});
  if(path==='/tasks/run'){
   const body=JSON.parse(options.body),index=requests.length;requests.push(body);
   assert.ok([...logs.values()].some(log=>log.some(e=>e.type==='analysis_requested'&&e.request_id===body.request_id)));
   if(index===unknownAt)throw Error('private-token-must-not-leak');return envelope({task_id:next++,status:'running'});
  }
  if(/^\/tasks\/\d+$/.test(path)){const id=Number(path.split('/').at(-1)),index=id-51;return envelope({task:{id,workflow_id:0,request_id:requests[index].request_id,status:index===failAt?'failed':'succeeded',steps:index===failAt?[]:[{index:0,tool:'kb_analyze',status:'succeeded',output:JSON.stringify(analysis)}]}});}
  throw Error('fixture未定义路径');
 };
 return {requests,calls,logs,events,args:{dataset:{version:1,documents:[source],cases},datasetSha256:'b'.repeat(64),kbMap:{project:7},context:context(),phase:'baseline',pollIntervalMs:0,
  createClient:()=>createApiClient({token:'fixture-access',fetchImpl}),
  openCase:async({index})=>{const log=[];logs.set(index,log);return {path:`tmp/fixture.case-${index}.jsonl`,append:async e=>{if(index===logFailAt&&e.type==='analysis_requested')throw Error('磁盘已满');log.push(e);},close:async()=>{}};},
  onManifest:async e=>events.push(e)}};
}
test('默认计划与输入边界不授权请求',()=>{const parse=feature('parseCohortArgs');assert.equal(parse([]).run,false);assert.throws(()=>parse(['--run']),/auth|认证/);assert.throws(()=>parse(['--base','https://example.test']),/本机/);assert.throws(()=>parse(['--plan','--run']),/混用|同时/);});
test('已知失败保留完整原UUID与失败链后继续，其余题每题只提交一次',async()=>{
 const f=fixture({failAt:1}),run=await feature('runCohort')(f.args);
 assert.equal(run.complete_cohort,true);assert.equal(run.counts.total,3);assert.equal(run.counts.succeeded,2);assert.equal(run.counts.failed,1);assert.equal(run.counts.unknown,0);
 assert.equal(f.requests.length,3);assert.equal(new Set(f.requests.map(x=>x.request_id)).size,3);assert.equal(run.successful_quality.n,2);assert.equal(run.usage.known_total_tokens,30);assert.equal(run.usage.unknown_case_ids.length,1);
 assert.equal(run.cases[1].status,'failed');assert.ok(run.cases[1].records.some(e=>e.type==='analysis_terminal'&&e.status==='failed'));
 assert.equal(feature('inspectCohort')(run,{dataset:f.args.dataset,datasetSha256:f.args.datasetSha256}).valid,true);
});
test('提交结果未知立即停止，不重发或继续下一道付费题',async()=>{
 const f=fixture({unknownAt:1}),run=await feature('runCohort')(f.args);assert.equal(run.complete_cohort,false);assert.equal(f.requests.length,2);assert.equal(run.counts.unknown,1);assert.equal(run.counts.not_started,1);assert.ok(run.cases[1].request_id);assert.equal(run.cases[1].task_id,0);assert.ok(!JSON.stringify(run).includes('private-token-must-not-leak'));
});
test('日志写失败禁止本题POST与后续付费调用',async()=>{
 const f=fixture({logFailAt:1});await assert.rejects(feature('runCohort')(f.args),/日志|磁盘/);assert.equal(f.requests.length,1);
});
test('完整比较按全体执行率与共同成功分母，拒绝数据篡改/不同配置',async()=>{
 const f=fixture({failAt:1}),before=await feature('runCohort')(f.args),g=fixture(),after=await feature('runCohort')({...g.args,phase:'candidate',context:{...context(),binary_sha:'c'.repeat(64)}}),validation={dataset:f.args.dataset,datasetSha256:f.args.datasetSha256};
 const result=feature('compareCohorts')(before,after,validation);assert.equal(result.comparable,true);assert.equal(result.paired_successful.n,2);assert.equal(result.baseline.counts.total,3);assert.equal(result.candidate.counts.succeeded,3);
 const changed=structuredClone(after);changed.context.config.return_top_k=6;assert.equal(feature('compareCohorts')(before,changed,validation).comparable,false);
 const bad=structuredClone(before);bad.cases[0].records.find(e=>e.type==='result').analysis.analysis='篡改后没有事实';assert.equal(feature('inspectCohort')(bad,validation).valid,false);
 const forged=structuredClone(before);forged.cases[1].records.find(e=>e.type==='analysis_terminal').task_id=999;assert.equal(feature('inspectCohort')(forged,validation).valid,false);
});
test('整轮结束语料漂移保留结果但不允许比较',async()=>{
 const f=fixture({drift:true}),run=await feature('runCohort')(f.args);assert.equal(run.complete_cohort,false);assert.equal(run.corpus_verified,false);assert.equal(f.requests.length,3);
});
test('cohort原始首尾与中间事件不可伪造',async()=>{
 const f=fixture(),run=await feature('runCohort')(f.args),parse=feature('parseCohortRecords');assert.equal(parse(f.events).complete_cohort,true);
 for(const mutate of [rows=>{rows[0].phase='candidate';},rows=>{rows[0].dataset_sha256='f'.repeat(64);},rows=>{rows[2].type='unknown';},rows=>{rows.splice(2,0,structuredClone(rows[2]));}]){const rows=structuredClone(f.events);mutate(rows);assert.throws(()=>parse(rows),/首尾|事件|数量/);}
 const bad=structuredClone(run);bad.http.total+=1;bad.http.get+=1;assert.equal(feature('inspectCohort')(bad,{dataset:f.args.dataset,datasetSha256:f.args.datasetSha256}).valid,false);
});
async function folderWork(run){const base=await realpath(resolve('tmp')),folder=await mkdtemp(join(base,'evidence-cohort-test-'));try{return await run(folder);}finally{const actual=await realpath(folder);assert.equal(dirname(actual),base);assert.ok(basename(actual).startsWith('evidence-cohort-test-'));await rm(actual,{recursive:true});}}
test('逐题使用真实幂等close，关闭失败不会伪装为已关闭',async()=>folderWork(async folder=>{
 const closed=[],bundle=await feature('reserveCohortOutputs')(join(folder,'run.jsonl'),2,{openLog:async path=>({append:async()=>{},close:async()=>{closed.push(path);if(path.endsWith('002.jsonl'))throw Error('fixture-close-failed');}})});
 await bundle.cases[0].close();await bundle.cases[0].close();assert.equal(closed.length,1);await assert.rejects(bundle.cases[1].close(),/关闭/);await assert.rejects(bundle.closeAll(),/关闭/);assert.equal(closed.filter(path=>path.endsWith('001.jsonl')).length,1);
}));
test('默认及显式plan在fetch被阻止时仍为0HTTP，预留冲突先于任何请求',async()=>folderWork(async folder=>{
 const preloadPath=join(folder,'blocked.mjs');await writeFile(preloadPath,"globalThis.fetch=()=>{throw Error('NO_HTTP_EXPECTED')};\n");const preload=pathToFileURL(preloadPath).href,execute=promisify(execFile);
 for(const args of [[],['--plan']]){const {stdout}=await execute(process.execPath,['--import',preload,'go-zero-admin/data/scripts/eval-kb-cohort.mjs',...args]);const plan=JSON.parse(stdout);assert.equal(plan.http_requests,0);assert.equal(plan.case_count,24);}
 const auth=join(folder,'auth.json'),map=join(folder,'map.json'),ctx=join(folder,'context.json');await writeFile(auth,JSON.stringify({access_token:'fixture-access',expires_at:Math.floor(Date.now()/1000)+3600}));await writeFile(map,JSON.stringify({project_ops:5,project_ai:6}));await writeFile(ctx,JSON.stringify(context()));
 const out=join(folder,'existing.jsonl');await writeFile(out.replace('.jsonl','.case-002.jsonl'),'keep-original');
 await assert.rejects(execute(process.execPath,['--import',preload,'go-zero-admin/data/scripts/eval-kb-cohort.mjs','--run','--auth-file',auth,'--kb-map',map,'--context-file',ctx,'--output',out]),error=>error.stderr.includes('输出已存在')&&!error.stderr.includes('NO_HTTP_EXPECTED'));
 assert.equal(await readFile(out.replace('.jsonl','.case-002.jsonl'),'utf8'),'keep-original');
}));
