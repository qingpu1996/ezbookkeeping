<template>
 <section class="asset-definitions">
  <header><div><h1>资产定义</h1><p>管理资产名称、数量单位与精度；买卖在“添加交易 → 支出 → 投资”中登记。</p></div><button class="primary" @click="fresh">＋ 添加</button></header>
  <div class="toolbar"><label class="search">搜索资产<input v-model="search" type="search" placeholder="名称、单位" /></label><label>使用状态<select v-model="status"><option value="all">全部</option><option value="used">已使用</option><option value="unused">未使用</option></select></label><button :disabled="store.loading" @click="reload">{{store.loading?'刷新中…':'刷新'}}</button></div>
  <p v-if="store.error" role="alert">{{store.error}}</p><p v-if="message" role="status">{{message}}</p>
  <p v-if="!store.loaded && store.loading">正在读取资产定义…</p>
  <div class="table-scroll"><table><thead><tr><th>资产</th><th>单位</th><th>数量精度</th><th>状态</th><th>操作</th></tr></thead><tbody>
   <tr v-for="d in visible" :key="d.id"><td><i class="las la-coins" aria-hidden="true" /> <strong>{{d.name}}</strong></td><td>{{d.unitName}} / {{d.unit}}</td><td>最多 {{d.precision}} 位小数</td><td><span class="badge">{{d.inUse?'已使用':'未使用'}}</span></td><td><button @click="edit(d)">编辑</button></td></tr>
   <tr v-if="store.loaded && !filtered.length"><td colspan="5">{{search || status!=='all'?'没有匹配的资产':'还没有资产定义，点击添加开始。'}}</td></tr>
  </tbody></table></div>
  <footer><span>共 {{filtered.length}} 项 · 第 {{page}} / {{pages}} 页</span><div><button :disabled="page<=1" @click="page--">上一页</button><button :disabled="page>=pages" @click="page++">下一页</button></div></footer>
  <p class="help">当前支持黄金规则。已用于持仓的单位和精度会锁定；修改名称不改变历史交易。</p>
  <dialog ref="editor" @cancel="cancelDialog"><form v-if="form" @submit.prevent="save"><header><h2>{{form.version ? '编辑资产定义':'资产定义设置'}}</h2><button type="button" :disabled="busy" @click="close">关闭</button></header>
   <fieldset :disabled="busy">
    <label>资产名称<input v-model="form.name" maxlength="128" required /></label>
    <label>资产规则<select v-model="form.kind" :disabled="form.inUse"><option value="gold">黄金</option></select></label>
    <div class="units"><label>单位名称<input v-model="form.unitName" maxlength="32" :disabled="form.inUse" required placeholder="克" /></label><label>单位符号<input v-model="form.unit" maxlength="16" :disabled="form.inUse" pattern="[A-Za-z][A-Za-z0-9_]{0,15}" required placeholder="g" /></label></div>
    <label>数量最多保留的小数位<input type="number" v-model.number="form.precision" min="0" max="12" :disabled="form.inUse" required /></label>
    <p>{{form.inUse?'已有持仓使用此定义，单位和精度不可修改；名称仍可修改。':'单位不自动换算；目前自动报价只适用于黄金 g 单位。'}}</p>
    <p v-if="error" role="alert">{{error}}</p>
    <footer><button type="button" @click="close">取消</button><button type="submit" class="primary">{{busy?'保存中…':'保存'}}</button></footer>
   </fieldset>
  </form></dialog>
 </section>
</template>
<script setup lang="ts">
import {ref,computed,watch,onMounted} from 'vue';
import {useInvestmentDefinitionsStore} from '@/stores/investmentDefinition';
import type {InvestmentDefinition} from '@/models/investment';
const store=useInvestmentDefinitionsStore(),form=ref<InvestmentDefinition>(),editor=ref<HTMLDialogElement>(),busy=ref(false),error=ref(''),message=ref(''),search=ref(''),status=ref('all'),page=ref(1);
const filtered=computed(()=>store.rows.filter(d=>(status.value==='all'||d.inUse===(status.value==='used')) && `${d.name} ${d.unitName} ${d.unit}`.toLocaleLowerCase().includes(search.value.trim().toLocaleLowerCase())).sort((a,b)=>a.name.localeCompare(b.name)));
const pages=computed(()=>Math.max(1,Math.ceil(filtered.value.length/15))),visible=computed(()=>filtered.value.slice((page.value-1)*15,page.value*15));
watch([search,status],()=>page.value=1);watch(pages,n=>page.value=Math.min(page.value,n));
function edit(d:InvestmentDefinition){form.value={...d};error.value='';editor.value?.showModal();}
function fresh(){edit({id:crypto.randomUUID(),name:'',kind:'gold',unit:'g',unitName:'克',precision:4,version:0,inUse:false});}
function close(){if(!busy.value)editor.value?.close();}
function cancelDialog(e:Event){if(busy.value)e.preventDefault();}
async function reload(){try{await store.load(true)}catch{/* Inline error retains cached list. */}}
async function save(){if(!form.value||busy.value)return;busy.value=true;error.value='';try{await store.save({...form.value});editor.value?.close();message.value='资产定义已保存，持仓和交易未改变。';}catch{error.value='保存失败。定义可能已更新或已被持仓使用，请关闭编辑窗口、刷新列表后重试。';}finally{busy.value=false;}}
onMounted(()=>{void store.load().catch(()=>{});});
</script>
<style scoped>
.asset-definitions{padding:24px;max-width:1200px;margin:auto}header,footer,.toolbar,.units{display:flex;align-items:center;gap:16px;justify-content:space-between}h1{font-size:24px}h2{font-size:20px}p{line-height:1.6;margin:12px 0}.toolbar{margin:24px 0;justify-content:flex-start;align-items:flex-end}.search{flex:1;max-width:420px}label{display:grid;gap:6px}input,select,button{font:inherit;color:inherit;background:transparent;border:1px solid #8886;border-radius:6px;padding:10px 12px;line-height:1.5}button{cursor:pointer;white-space:nowrap}.primary{background:#bd7b44;color:white;border-color:transparent}button:disabled,input:disabled{opacity:.5}.table-scroll{overflow:auto}table{border-collapse:collapse;width:100%;text-align:left}th,td{padding:14px 12px;border-bottom:1px solid #8883;white-space:nowrap}th{font-weight:500;opacity:.7}.badge{background:#8881;border-radius:5px;padding:5px 8px}.las{font-size:24px;color:#b68131;vertical-align:middle}footer{margin-top:20px}footer button+button{margin-left:8px}.help{font-size:13px;opacity:.7}dialog{width:min(560px,calc(100vw - 32px));padding:24px;border:1px solid #8884;border-radius:12px;background:var(--f7-list-bg-color,#fff);color:inherit;margin:auto;max-height:90vh;overflow:auto}dialog::backdrop{background:#0007}fieldset{border:0;padding:0}fieldset>label,.units{margin:16px 0}.units label{min-width:0;flex:1}.units input{width:100%;box-sizing:border-box}[role=alert]{color:#b44228}@media(max-width:600px){.asset-definitions{padding:16px}.toolbar{flex-wrap:wrap}.search{min-width:100%}header{align-items:flex-start}header p{font-size:13px}td,th{padding:12px 8px}dialog{box-sizing:border-box}}
</style>

<style scoped>
:global(.v-theme--dark) dialog {background:#2f3349;}
</style>
