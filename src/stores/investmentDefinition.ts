import {ref} from 'vue';
import {defineStore} from 'pinia';
import services from '@/lib/services';
import type {InvestmentDefinition} from '@/models/investment';
export const useInvestmentDefinitionsStore=defineStore('investmentDefinitions',()=>{
 const rows=ref<InvestmentDefinition[]>([]),loaded=ref(false),loading=ref(false),error=ref('');
 let generation=0,flight:Promise<void>|undefined,last=0;
 function reset(){generation++;rows.value=[];loaded.value=false;loading.value=false;error.value='';flight=undefined;last=0;}
 function load(force=false):Promise<void>{
  if(flight)return flight;
  if(!force && loaded.value && Date.now()-last<60000)return Promise.resolve();
  const g=generation;loading.value=true;error.value='';
  const request=services.getInvestmentDefinitions().then(r=>{if(g===generation){rows.value=r.data.result;loaded.value=true;last=Date.now();}}).catch(e=>{if(g===generation)error.value='刷新失败，当前列表可能不是最新。请重试。';throw e}).finally(()=>{if(g===generation){loading.value=false;flight=undefined;}});
  flight=request;return request;
 }
 async function save(d:InvestmentDefinition){const g=generation;const r=await services.saveInvestmentDefinition(d);if(g!==generation)throw Error('登录状态已改变');rows.value=rows.value.filter(x=>x.id!==r.data.result.id).concat(r.data.result);last=0;return r.data.result;}
 return {rows,loaded,loading,error,load,save,reset};
});
