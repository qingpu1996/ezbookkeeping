<template>
 <section class="asset-definitions">
  <h1>资产定义</h1><p>先定义资产名称和数量单位，再将它关联到持仓账户。这里不记录买卖。</p>
  <p>当前支持黄金的数量与成本记账。其他资产的专属业务规则尚未开放。</p>
  <p v-if="error" role="alert">{{ error }}</p><p v-if="message" role="status">{{ message }}</p>
  <div class="definitions"><button v-for="d in rows" :key="d.id" :disabled="busy" @click="edit(d)"><strong>{{d.name}}</strong><span>{{d.unitName}} / {{d.unit}} · 最多 {{d.precision}} 位小数{{d.inUse?' · 已使用':''}}</span></button></div>
  <button :disabled="busy" @click="fresh">添加资产定义</button>
  <form v-if="form" @submit.prevent="save"><fieldset :disabled="busy">
   <label>资产名称<input v-model="form.name" maxlength="128" required /></label>
   <label>资产规则<select v-model="form.kind"><option value="gold">黄金</option></select></label>
   <label>单位名称<input v-model="form.unitName" maxlength="32" :disabled="form.inUse" required placeholder="例如：克" /></label>
   <label>单位符号<input v-model="form.unit" maxlength="16" :disabled="form.inUse" pattern="[A-Za-z][A-Za-z0-9_]{0,15}" required placeholder="例如：g" /></label>
   <label>数量最多保留的小数位<input type="number" v-model.number="form.precision" min="0" max="12" :disabled="form.inUse" required /></label>
   <p v-if="form.inUse">已被持仓使用，单位和精度已锁定；名称仍可修改。</p>
   <p v-else>单位不自动换算。请在关联账户前确认；目前自动报价只适用于黄金 g 单位。</p>
   <button type="submit">保存资产定义</button>
  </fieldset></form>
 </section>
</template>
<script setup lang="ts">
import {ref,onMounted} from 'vue';
import services from '@/lib/services';
import type {InvestmentDefinition} from '@/models/investment';
const rows=ref<InvestmentDefinition[]>([]),form=ref<InvestmentDefinition>(),busy=ref(false),error=ref(''),message=ref('');
async function load(){rows.value=(await services.getInvestmentDefinitions()).data.result;}
function edit(d:InvestmentDefinition){form.value={...d};message.value='';}
function fresh(){form.value={id:crypto.randomUUID(),name:'',kind:'gold',unit:'g',unitName:'克',precision:4,version:0,inUse:false};message.value='';}
async function save(){if(!form.value||busy.value)return;busy.value=true;error.value='';try{const r=await services.saveInvestmentDefinition(form.value);form.value=r.data.result;await load();message.value='资产定义已保存，没有修改任何持仓或交易。';}catch{error.value='未能保存，请重新选择该定义核对最新设置后重试。已使用的单位和精度不能修改。';}finally{busy.value=false;}}
onMounted(async()=>{busy.value=true;try{await load();}catch{error.value='读取资产定义失败，请刷新重试。';}finally{busy.value=false;}});
</script>
<style scoped>
.asset-definitions{max-width:850px;margin:auto;padding:24px}h1{font-size:24px}p{line-height:1.7;opacity:.8}.definitions{display:flex;gap:12px;flex-wrap:wrap;margin:20px 0}.definitions button{display:grid;gap:8px;text-align:left}form{max-width:540px;margin:24px 0}fieldset{border:0;padding:0}label{display:grid;gap:8px;margin:16px 0}input,select,button{font:inherit;color:inherit;background:transparent;border:1px solid #8886;border-radius:8px;padding:12px}button{cursor:pointer}button:disabled,input:disabled{opacity:.5}[role=alert]{color:#ac442d}
</style>
