import {createHash} from 'node:crypto';
import {readFile, realpath, lstat} from 'node:fs/promises';
import {dirname, resolve, relative, isAbsolute} from 'node:path';
import {fileURLToPath, pathToFileURL} from 'node:url';
import {createApiClient, loadDataset, runEvaluation, listDocuments, inventoryOf, readBoundDocument, matchSourceDocuments,
  openReport, summarize, compareRuns, validateContext} from './eval-kb-quality.mjs';
import {validateHits} from './eval-kb-recall.mjs';
import {evaluateRetrieval} from './eval-kb-quality-metrics.mjs';

const root=resolve(dirname(fileURLToPath(import.meta.url)),'../../..');
const hash=value=>createHash('sha256').update(value).digest('hex');
const positive=value=>Number.isSafeInteger(value)&&value>0;
const uuid=value=>typeof value==='string'&&/^[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/iu.test(value);
const canonical=value=>Array.isArray(value)?value.map(canonical):value&&typeof value==='object'?Object.fromEntries(Object.keys(value).sort().map(key=>[key,canonical(value[key])])):value;
const stable=value=>JSON.stringify(canonical(value));
const equal=(a,b)=>stable(a)===stable(b);
const require=(value,message)=>{if(!value)throw Error(message);};
const inside=(base,path)=>{const part=relative(base,path);return !!part&&!isAbsolute(part)&&part!=='..'&&!part.startsWith('../')&&!part.startsWith('..\\');};
const fixedContext=context=>Object.fromEntries(Object.entries(context).filter(([key])=>!['binary_sha','source_revision','experiment'].includes(key)));
const httpKeys=['total','get','post','search','analysis_submissions','task_reads','failed'];
const httpTotal=clients=>Object.fromEntries(httpKeys.map(key=>[key,clients.reduce((n,client)=>n+(client.counts?.[key]??0),0)]));

export function parseCohortArgs(args){
 const options={run:false,phase:'baseline',base:'http://127.0.0.1:7001',cases:resolve(root,'ai_document/ai-rag/business-evidence-cases.json')};
 const values={'--cases':'cases','--auth-file':'authFile','--kb-map':'kbMapFile','--context-file':'contextFile','--output':'output','--base':'base','--phase':'phase','--report':'report'};
 const seen=new Set();
 for(let index=0;index<args.length;index++){
  const key=args[index];require(!seen.has(key),'参数重复');seen.add(key);
  if(key==='--plan')options.plan=true;
  else if(key==='--run')options.run=true;
  else if(key==='--compare'){require(args[index+1]&&args[index+2]&&!args[index+1].startsWith('--')&&!args[index+2].startsWith('--'),'--compare需要两份cohort日志');options.compare=[args[++index],args[++index]];}
  else{require(values[key]&&args[index+1]&&!args[index+1].startsWith('--'),'未知参数或参数缺值');options[values[key]]=args[++index];}
 }
 require([options.run,options.plan,!!options.report,!!options.compare].filter(Boolean).length<=1,'--plan/--run/--report/--compare不能混用');
 require(['baseline','candidate'].includes(options.phase),'phase仅支持baseline/candidate');
 let url;try{url=new URL(options.base);}catch{throw Error('base不是本机HTTP根地址');}
 require(url.protocol==='http:'&&['127.0.0.1','localhost','[::1]'].includes(url.hostname)&&!url.username&&!url.password&&url.pathname==='/'&&!url.search&&!url.hash,'base仅允许本机HTTP根地址');options.base=url.origin;
 if(options.run)require(options.authFile&&options.kbMapFile&&options.contextFile&&options.output,'--run需要auth-file、kb-map、context-file和新output');
 if(options.output){options.output=resolve(options.output);require(inside(resolve(root,'tmp'),options.output)&&options.output.endsWith('.jsonl'),'output必须为仓库tmp内新.jsonl');}
 return options;
}

function validateInput({dataset,datasetSha256,kbMap,context,phase}){
 validateContext(context);require(/^[a-f0-9]{64}$/u.test(datasetSha256),'题集摘要无效');require(['baseline','candidate'].includes(phase),'phase无效');
 require(dataset?.version===1&&Array.isArray(dataset.documents)&&dataset.documents.length&&Array.isArray(dataset.cases)&&dataset.cases.length&&dataset.cases.length<=100,'题集结构无效');
 const ids=dataset.cases.map(item=>item.id),keys=[...new Set(dataset.cases.map(item=>item.kb))];
 require(ids.every(id=>typeof id==='string'&&id)&&new Set(ids).size===ids.length,'题ID缺失或重复');
 require(kbMap&&keys.every(key=>positive(kbMap[key]))&&new Set(keys.map(key=>kbMap[key])).size===keys.length,'知识库映射缺失或重复');
}
async function collectCorpus(dataset,kbMap,client){
 const keys=[...new Set(dataset.cases.map(item=>item.kb))],known=await listDocuments(kbMap,keys,client),documents=[];
 for(const source of dataset.documents.filter(doc=>keys.includes(doc.kb))){const matches=matchSourceDocuments(source,[...known.get(source.kb).values()]);require(matches.length===1,'文档ID/版本或标题不唯一/缺失');documents.push(await readBoundDocument(source,matches[0],client));}
 return {documents,inventory:inventoryOf(known)};
}
function validateCorpus(corpus,dataset,kbMap){
 const groups=new Set(dataset.cases.map(item=>item.kb)),expected=dataset.documents.filter(doc=>groups.has(doc.kb));
 require(corpus&&Array.isArray(corpus.documents)&&Array.isArray(corpus.inventory)&&corpus.documents.length===expected.length,'整轮语料结构或数量不符');
 require(new Set(corpus.documents.map(doc=>doc.key)).size===expected.length&&new Set(corpus.inventory.map(doc=>doc.id)).size===corpus.inventory.length,'整轮语料或库存重复');
 for(const source of expected){
  const bound=corpus.documents.find(doc=>doc.key===source.key);
  require(bound&&Object.entries(source).every(([key,value])=>equal(bound[key],value))&&bound.content_sha256===source.sha256&&positive(bound.doc_id)&&bound.kb_id===kbMap[source.kb]&&positive(bound.version)&&bound.status==='ready'&&typeof bound.embedding_model==='string'&&bound.embedding_model.trim(),'整轮文档与冻结正文/版本不符');
  require(corpus.inventory.some(doc=>doc.id===bound.doc_id&&doc.kb===bound.kb&&doc.kb_id===bound.kb_id&&doc.title===bound.title&&doc.version===bound.version&&doc.status===bound.status&&doc.embedding_model===bound.embedding_model),'绑定文档缺少对应库存');
 }
 require(corpus.inventory.every(doc=>positive(doc.id)&&positive(doc.version)&&groups.has(doc.kb)&&doc.kb_id===kbMap[doc.kb]&&typeof doc.title==='string'&&doc.title.trim()&&['ready','pending','failed'].includes(doc.status)&&typeof doc.embedding_model==='string'),'整轮库存归属或字段无效');
}
function validateHTTP(value){require(value&&httpKeys.every(key=>Number.isSafeInteger(value[key])&&value[key]>=0)&&value.total===value.get+value.post,'HTTP口径不完整');}
function validateFailure(records,item,meta,corpus){
 const types=['meta','bindings','case_started','search_result','analysis_requested','analysis_accepted','analysis_terminal','error'];
 require(records.length===types.length&&records.every((event,i)=>event.type===types[i]),'失败记录链不完整或含未知事件');
 const [first,bindings,started,search,requested,accepted,terminal,last]=records,kbID=meta.kb_map[item.kb];
 require(first.version===1&&first.dataset_sha256===meta.dataset_sha256&&equal(first.case_ids,[item.id])&&equal(first.kb_map,meta.kb_map)&&equal(first.context,meta.context)&&first.phase===meta.phase&&first.split===item.split,'失败记录meta与整轮不符');
 require(equal(bindings.documents,corpus.documents.filter(doc=>doc.kb===item.kb))&&equal(bindings.inventory,corpus.inventory.filter(doc=>doc.kb===item.kb)),'失败记录语料绑定不符');
 require([started,search,requested,accepted,terminal].every(event=>event.case_id===item.id)&&started.kb_id===kbID&&started.split===item.split&&started.category===item.category&&search.kb_id===kbID&&search.query===item.query,'失败题问项或归属不符');
 const known=new Map(bindings.inventory.map(doc=>[doc.id,doc])),bound=new Map(bindings.documents.map(doc=>[doc.key,doc]));validateHits(search.list,kbID,known);
 require(typeof search.search_client_ms==='number'&&Number.isFinite(search.search_client_ms)&&search.search_client_ms>=0&&Number.isSafeInteger(search.search_server_ms)&&search.search_server_ms>=0&&equal(search.retrieval,evaluateRetrieval(item,search.list,bound)),'失败题检索观测或派生结果不符');
 require(uuid(requested.request_id)&&requested.payload?.request_id===requested.request_id&&typeof requested.payload.name==='string'&&requested.payload.name===`eval-kb-quality-${item.id.slice(0,40)}-${requested.request_id}`&&requested.payload.input==='{}','失败题原UUID/载荷不符');
 let definition;try{definition=JSON.parse(requested.payload.definition);}catch{throw Error('失败题DSL损坏');}
 require(equal(definition,{steps:[{name:'知识库分析',tool:'kb_analyze',input:{kb_id:kbID,query:item.query}}]}),'失败题不是冻结查询');
 require(accepted.request_id===requested.request_id&&positive(accepted.task_id)&&['pending','running','succeeded'].includes(accepted.status)&&terminal.request_id===requested.request_id&&terminal.task_id===accepted.task_id&&['failed','rejected','cancelled'].includes(terminal.status),'失败题receipt或已知终态不符');
 require(last.case_id===item.id&&last.retry===false&&last.comparable===false,'失败记录未声明禁止重试');
 return {request_id:requested.request_id,task_id:accepted.task_id,status:terminal.status};
}
function checkedSuccess(records,dataset,datasetSha256){
 const before=structuredClone(records),after=structuredClone(records);before[0].phase='baseline';after[0].phase='candidate';
 const checked=compareRuns(before,after,{dataset,datasetSha256});require(checked.comparable,'成功记录未通过原v1完整链与派生复算核对');
 return records.find(event=>event.type==='result');
}
function summarizeCases(rows,total){
 const successful=rows.filter(row=>row.status==='succeeded'),results=successful.map(row=>row.records.find(event=>event.type==='result'));
 const counts={total,attempted:rows.length,succeeded:successful.length,failed:rows.filter(row=>row.status==='failed').length,unknown:rows.filter(row=>row.status==='unknown').length,not_started:total-rows.length};
 const successfulQuality=summarize(results),positives=results.filter(row=>row.retrieval.kind==='positive'),matched=positives.flatMap(row=>row.answer_screen.cited_evidence_groups_matched??[]),facts=positives.flatMap(row=>row.answer_screen.required_groups_matched??[]);
 return {counts,execution_success_rate:total?counts.succeeded/total:null,successful_quality:successfulQuality,
  cited_evidence:{positive_n:positives.length,covered:matched.filter(Boolean).length,total:matched.length,coverage:matched.length?matched.filter(Boolean).length/matched.length:null,complete_cases:positives.filter(row=>row.answer_screen.cited_evidence_groups_matched.every(Boolean)).length},
  answer_facts:{positive_n:positives.length,covered:facts.filter(Boolean).length,total:facts.length,coverage:facts.length?facts.filter(Boolean).length/facts.length:null,complete_cases:positives.filter(row=>row.answer_screen.required_groups_matched.every(Boolean)).length},
  usage:{...successfulQuality.usage,unknown_case_ids:rows.filter(row=>row.status!=='succeeded'||!row.records.find(event=>event.type==='result')?.analysis.usage.complete).map(row=>row.case_id),note:'仅成功输出保存的已知聊天用量；失败/未知不能按零计费，不含Embedding。'}};
}

export async function runCohort({dataset,datasetSha256,kbMap,context,phase='baseline',createClient,openCase,onManifest=async()=>{},pollIntervalMs=750,taskTimeoutMs=100_000,now}){
 validateInput({dataset,datasetSha256,kbMap,context,phase});require(typeof createClient==='function'&&typeof openCase==='function','缺少client或日志工厂');
 const meta={version:1,phase,dataset_sha256:datasetSha256,case_ids:dataset.cases.map(item=>item.id),kb_map:kbMap,context};
 const clients=[],newClient=()=>{const client=createClient();clients.push(client);return client;};
 const corpusClient=newClient(),before=await collectCorpus(dataset,kbMap,corpusClient),rows=[];validateCorpus(before,dataset,kbMap);
 await onManifest({type:'cohort_meta',...meta,started_at:new Date().toISOString()});await onManifest({type:'cohort_corpus_before',...before});
 let stopped=false;
 for(const [index,item] of dataset.cases.entries()){
  const log=await openCase({index,case_id:item.id}),records=[],client=newClient();let persistenceFailure;
  const append=async event=>{try{await log.append(event);records.push(event);}catch(error){persistenceFailure=error;throw Error('逐题日志持久化失败，停止后续调用');}};
  const first={type:'meta',...meta,split:item.split,case_ids:[item.id],started_at:new Date().toISOString()};
  let row;
  try{
   await append(first);
   const result=await runEvaluation({dataset:{...dataset,cases:[item]},kbMap,client,pollIntervalMs,taskTimeoutMs,...(now?{now}:{}),onProgress:async event=>{
    if(event.type==='bindings')require(equal(event.documents,before.documents.filter(doc=>doc.kb===item.kb))&&equal(event.inventory,before.inventory.filter(doc=>doc.kb===item.kb)),'逐题开始前语料与整轮绑定不同');
    await append(event);
   }});
   await append({type:'summary',completed:true,completed_at:new Date().toISOString(),...result.summary});
   const actual=checkedSuccess(records,dataset,datasetSha256);row={case_id:item.id,status:'succeeded',request_id:actual.request_id,task_id:actual.task_id,path:log.path,records};
  }catch(error){
   if(persistenceFailure)throw Error('逐题日志持久化失败，停止后续调用');
   await append({type:'error',case_id:item.id,stopped_at:new Date().toISOString(),message:error.message,retry:false,comparable:false,calls:{...client.counts}});
   let known;try{known=validateFailure(records,item,meta,before);}catch{/* 未知或结构异常保留原标识，整轮停止。 */}
   const requested=records.find(event=>event.type==='analysis_requested'),accepted=records.find(event=>event.type==='analysis_accepted');
   row={case_id:item.id,status:known?'failed':'unknown',terminal_status:known?.status??null,request_id:requested?.request_id??'',task_id:accepted?.task_id??0,path:log.path,records};
   if(!known)stopped=true;
  }finally{try{await log.close();}catch{throw Error('逐题日志关闭失败，停止后续调用');}}
  row.http={...client.counts};row.records_sha256=hash(records.map(event=>JSON.stringify(event)+'\n').join(''));rows.push(row);await onManifest({type:'cohort_case',...row});if(stopped)break;
 }
 let after=null,verified=false,error='';
 try{after=await collectCorpus(dataset,kbMap,corpusClient);verified=equal(before,after);if(!verified)error='整轮结束语料库存或版本不同';}catch{error='整轮结束语料复核失败';}
 const report={type:'cohort_summary',...meta,complete_cohort:!stopped&&rows.length===dataset.cases.length&&verified,corpus_verified:verified,before,after,cases:rows,...summarizeCases(rows,dataset.cases.length),corpus_http:{...corpusClient.counts},http:httpTotal(clients),error,
  note:'完整题集包含已知失败；质量指标分母只含成功题。失败/未知不计为空召回或错误答案，不自动重放；实际供应商尝试次数未知。'};
 await onManifest(report);return report;
}

export function inspectCohort(report,{dataset,datasetSha256}={}){
 try{
  validateInput({dataset,datasetSha256,kbMap:report.kb_map,context:report.context,phase:report.phase});
  require(report.type==='cohort_summary'&&report.version===1&&report.dataset_sha256===datasetSha256&&equal(report.case_ids,dataset.cases.map(item=>item.id)),'整轮题集或顺序不符');
  require(report.complete_cohort===true&&report.corpus_verified===true&&equal(report.before,report.after),'整轮未完成或语料未保持');
  validateCorpus(report.before,dataset,report.kb_map);
  require(Array.isArray(report.cases)&&report.cases.length===dataset.cases.length,'缺少已知终态题目');
  const requests=new Set(),tasks=new Set();
  for(const [i,row]of report.cases.entries()){
   const item=dataset.cases[i];require(row.case_id===item.id&&Array.isArray(row.records)&&row.records_sha256===hash(row.records.map(event=>JSON.stringify(event)+'\n').join('')),'逐题原始记录摘要或顺序不符');
   require(uuid(row.request_id)&&positive(row.task_id)&&!requests.has(row.request_id)&&!tasks.has(row.task_id),'逐题标识缺失或重复');requests.add(row.request_id);tasks.add(row.task_id);
   require(equal(row.records[0].context,report.context)&&equal(row.records[0].kb_map,report.kb_map)&&row.records[0].phase===report.phase,'逐题上下文与整轮不符');
   const binding=row.records.find(event=>event.type==='bindings');require(binding&&equal(binding.documents,report.before.documents.filter(doc=>doc.kb===item.kb))&&equal(binding.inventory,report.before.inventory.filter(doc=>doc.kb===item.kb)),'逐题语料与整轮不符');
   validateHTTP(row.http);const recordedHTTP=row.records.at(-1).calls?.http??row.records.at(-1).calls;require(equal(row.http,recordedHTTP),'逐题HTTP与原始记录不符');
   require(row.http.search===1&&row.http.analysis_submissions===1&&row.http.post===2&&row.http.failed===0&&row.http.task_reads>=1&&row.http.get>=row.http.task_reads+binding.documents.length+1,'完整逐题调用数与执行链不符');
   if(row.status==='succeeded'){const actual=checkedSuccess(row.records,dataset,datasetSha256);require(actual.request_id===row.request_id&&actual.task_id===row.task_id,'成功receipt不符');}
   else{require(row.status==='failed','包含未知题');const actual=validateFailure(row.records,item,report,report.before);require(actual.request_id===row.request_id&&actual.task_id===row.task_id&&actual.status===row.terminal_status,'失败receipt不符');}
  }
  const recomputed=summarizeCases(report.cases,dataset.cases.length);
  for(const key of ['counts','execution_success_rate','successful_quality','cited_evidence','answer_facts','usage'])require(equal(report[key],recomputed[key]),'整轮派生指标与原始观测不符');
  validateHTTP(report.http);validateHTTP(report.corpus_http);require(report.corpus_http.post===0&&report.corpus_http.failed===0&&report.corpus_http.search===0&&report.corpus_http.analysis_submissions===0&&report.corpus_http.task_reads===0&&report.corpus_http.get>0,'语料复核HTTP含额外调用或失败');
  require(equal(report.http,httpTotal([{counts:report.corpus_http},...report.cases.map(row=>({counts:row.http}))])),'整轮HTTP与各阶段保存计数不符');
  return {valid:true,reasons:[],...recomputed};
 }catch(error){return {valid:false,reasons:[error.message]};}
}
export function compareCohorts(baseline,candidate,validation){
 const before=inspectCohort(baseline,validation),after=inspectCohort(candidate,validation),reasons=[...before.reasons.map(x=>'baseline：'+x),...after.reasons.map(x=>'candidate：'+x)];
 if(baseline.phase!=='baseline'||candidate.phase!=='candidate')reasons.push('需要baseline与candidate阶段');
 if(!equal(baseline.kb_map,candidate.kb_map)||!equal(baseline.case_ids,candidate.case_ids)||!equal(baseline.before,candidate.before))reasons.push('题目/知识库/语料库存不同');
 if(!equal(fixedContext(baseline.context??{}),fixedContext(candidate.context??{})))reasons.push('模型、Embedding、预算、TopK或其余上下文不同');
 if(reasons.length)return {comparable:false,reasons};
 const pairedIDs=baseline.cases.filter(row=>row.status==='succeeded'&&candidate.cases.some(other=>other.case_id===row.case_id&&other.status==='succeeded')).map(row=>row.case_id);
 const paired=report=>summarizeCases(report.cases.filter(row=>pairedIDs.includes(row.case_id)),pairedIDs.length);
 return {comparable:true,reasons:[],baseline:before,candidate:after,execution_success_rate_delta:after.execution_success_rate-before.execution_success_rate,
  paired_successful:{n:pairedIDs.length,case_ids:pairedIDs,baseline:paired(baseline),candidate:paired(candidate),note:'仅共同成功子集，不代表全体题目质量提升；全体执行结果见baseline/candidate.counts。'},
  allowed_changes:['binary_sha','source_revision','experiment'].filter(key=>!equal(baseline.context[key],candidate.context[key])).map(key=>({field:key,baseline:baseline.context[key],candidate:candidate.context[key]}))};
}
async function outputPaths(output,count){
 const parent=await realpath(dirname(output)),tmp=await realpath(resolve(root,'tmp'));require(parent===tmp||inside(tmp,parent),'输出父目录越出tmp');
 return [output,...Array.from({length:count},(_,i)=>output.replace(/\.jsonl$/u,`.case-${String(i+1).padStart(3,'0')}.jsonl`))];
}
export async function reserveCohortOutputs(output,count,{openLog=openReport}={}){
 require(Number.isSafeInteger(count)&&count>0&&count<=100,'输出题数必须为1..100');require(inside(resolve(root,'tmp'),resolve(output))&&output.endsWith('.jsonl'),'output必须为仓库tmp内新.jsonl');
 const paths=await outputPaths(output,count),logs=[];
 const closeAll=async()=>{const closed=await Promise.allSettled(logs.map(log=>log.close()));if(closed.some(item=>item.status==='rejected'))throw Error('cohort日志关闭失败，禁止视为完整');};
 try{
  for(const path of paths){
   try{const info=await lstat(path);if(info)throw Error('输出已存在，拒绝覆盖历史');}catch(error){if(error.code!=='ENOENT')throw error;}
   const log=await openLog(path);let state='open';
   logs.push({path,append:async value=>{require(state==='open','日志已关闭或不可写');await log.append(value);},close:async()=>{
    if(state==='closed')return;require(state==='open','cohort日志此前关闭失败');
    try{await log.close();state='closed';}catch{state='failed';throw Error('cohort日志关闭失败');}
   }});
  }
 }catch(error){try{await closeAll();}catch{/* 失败保留输出，不进行HTTP。 */}throw error;}
 return {manifest:logs[0],cases:logs.slice(1),paths,closeAll};
}
async function readJSON(path,label){try{return JSON.parse(await readFile(path,'utf8'));}catch{throw Error(label+'不可读或不是JSON');}}
async function readCohort(path){
 const bytes=await readFile(path,'utf8');require(bytes.endsWith('\n'),'cohort日志未完整落盘');return parseCohortRecords(bytes.trim().split('\n').map(JSON.parse));
}
export function parseCohortRecords(records){
 require(Array.isArray(records)&&records.length>=3,'cohort日志事件数量无效');const report=records.at(-1);
 require(records[0]?.type==='cohort_meta'&&records[1]?.type==='cohort_corpus_before'&&report?.type==='cohort_summary','cohort日志缺少完整边界');
 require(['version','phase','dataset_sha256','kb_map','context','case_ids'].every(key=>equal(records[0][key],report[key]))&&equal(records[1].documents,report.before?.documents)&&equal(records[1].inventory,report.before?.inventory),'cohort首尾上下文不同');
 require(Array.isArray(report.cases)&&records.length===report.cases.length+3&&records.slice(2,-1).every(row=>row.type==='cohort_case'),'cohort事件类型或数量无效');
 require(equal(records.slice(2,-1).map(({type,...row})=>row),report.cases),'cohort逐题日志与终态摘要不同');return report;
}
async function main(args){
 const options=parseCohortArgs(args),dataset=await loadDataset(options.cases),datasetSha256=hash(await readFile(options.cases)),validation={dataset,datasetSha256};
 if(options.report||options.compare){const report=options.compare?compareCohorts(await readCohort(options.compare[0]),await readCohort(options.compare[1]),validation):{report:await readCohort(options.report)};if(report.report)report.validation=inspectCohort(report.report,validation);console.log(JSON.stringify(report,null,2));if(report.comparable===false||report.validation?.valid===false)process.exitCode=1;return;}
 if(!options.run){console.log(JSON.stringify({mode:'plan',http_requests:0,dataset_sha256:datasetSha256,case_count:dataset.cases.length,case_ids:dataset.cases.map(item=>item.id),max_analysis_calls:dataset.cases.length,automatic_retries:0,note:'已知终态失败单列后继续；结果未知或日志失败停止；必须显式--run及本地认证配置。'},null,2));return;}
 const auth=await readJSON(options.authFile,'认证'),kbMap=await readJSON(options.kbMapFile,'知识库映射'),context=await readJSON(options.contextFile,'运行上下文');
 require(typeof auth.access_token==='string'&&auth.access_token.trim()&&(!auth.expires_at||auth.expires_at>Date.now()/1000),'认证缺失或已过期，请正常登录');validateInput({dataset,datasetSha256,kbMap,context,phase:options.phase});
 const outputs=await reserveCohortOutputs(options.output,dataset.cases.length);
 try{
  const report=await runCohort({dataset,datasetSha256,kbMap,context,phase:options.phase,createClient:()=>createApiClient({token:auth.access_token,base:options.base}),openCase:async({index})=>outputs.cases[index],onManifest:async event=>{await outputs.manifest.append(event);if(event.type==='cohort_case')console.log(JSON.stringify({case_id:event.case_id,status:event.status,request_id:event.request_id,task_id:event.task_id}));}});
  await outputs.closeAll();
  console.log(JSON.stringify({complete_cohort:report.complete_cohort,counts:report.counts,usage:report.usage,http:report.http,output:options.output}));if(!report.complete_cohort)process.exitCode=1;
 }finally{await outputs.closeAll();}
}
if(process.argv[1]&&pathToFileURL(resolve(process.argv[1])).href===import.meta.url)main(process.argv.slice(2)).catch(error=>{console.error(error.message);process.exitCode=1;});
