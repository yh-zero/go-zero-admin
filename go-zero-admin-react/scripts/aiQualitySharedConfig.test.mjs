import test from 'node:test';
import assert from 'node:assert/strict';
import {fileURLToPath} from 'node:url';
import {createServer,isFileServingAllowed,normalizePath} from 'vite';

test('质量评分共用模块可访问且未开放后端配置和源码目录',async()=>{
 const server=await createServer({configFile:'vite.config.ts',logLevel:'error',server:{middlewareMode:true,watch:null,ws:false}});
 try{
  const url=relative=>'/@fs/'+normalizePath(fileURLToPath(new URL(relative,import.meta.url)));
  assert.equal(isFileServingAllowed(server.config,url('../../go-zero-admin/data/scripts/eval-kb-quality-metrics.mjs')),true);
  assert.equal(isFileServingAllowed(server.config,url('../../go-zero-admin/data/scripts/eval-kb-quality-context.mjs')),true);
  assert.equal(isFileServingAllowed(server.config,url('../../go-zero-admin/data/scripts/kb-custom-dataset.mjs')),true);
  assert.equal(isFileServingAllowed(server.config,url('../../go-zero-admin/.env.local')),false);
  assert.equal(isFileServingAllowed(server.config,url('../../go-zero-admin/application/ai/rpc/internal/svc/service_context.go')),false);
 }finally{await server.close();}
});
