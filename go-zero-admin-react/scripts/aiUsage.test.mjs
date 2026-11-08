import test from 'node:test';
import assert from 'node:assert/strict';
import { createServer } from 'vite';

test('用量未知值、策略版本与会话隔离', async t => {
 const server=await createServer({configFile:false,envFile:false,logLevel:'error',server:{middlewareMode:true,watch:null,ws:false}});
 try {
  const state=await server.ssrLoadModule('/src/pages/ai/usageState.ts');
  await t.test('缺失用量不显示为零，明确上报的零保留',()=>{
   assert.equal(state.usageTokenLabel({usage_state:'unknown'}),'未取得');
   assert.equal(state.usageTokenLabel({usage_state:'complete',total_tokens:0}),'0');
   assert.equal(state.usageTokenLabel({usage_state:'partial',total_tokens:12}),'12（不完整）');
  });
  await t.test('策略保存使用读到的版本，全局不能继承自己',()=>{
   assert.deepEqual(state.usagePolicyPatch({owner_id:7,version:4},{inherit_default:true,daily_request_limit:2,daily_token_warning:0}),{expected_version:4,inherit_default:true,daily_token_warning:0});
   assert.throws(()=>state.usagePolicyPatch({owner_id:0,version:4},{inherit_default:true}),/全局/);
   assert.throws(()=>state.usagePolicyPatch({owner_id:7,version:4},{daily_request_limit:-1}),/额度/);
  });
  await t.test('身份、部门、本人和全局的查询键均隔离',()=>{
   const params={page:1};
   const a=state.usageQueryKey(['session',1,0],false,'calls',params);
   assert.notDeepEqual(a,state.usageQueryKey(['session',1,1],false,'calls',params));
   assert.notDeepEqual(a,state.usageQueryKey(['session',1,0],true,'calls',params));
  });
 } finally {await server.close();}
});
