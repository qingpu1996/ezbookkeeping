import {computed,reactive,ref,watch,type Ref} from 'vue';
import {useInvestmentDefinitionsStore} from '@/stores/investmentDefinition';
import type {RecognizedTransactionResponse} from '@/models/large_language_model';
import {assetRecognitionDraft} from '@/models/assetRecognition';
import services from '@/lib/services';
import {TransactionType} from '@/core/transaction';
import type {Transaction} from '@/models/transaction';
import type {TransactionTemplate} from '@/models/transaction_template';
import type {InvestmentDetail,InvestmentPosition,InvestmentRequest} from '@/models/investment';
import {investmentMinorUnits} from '@/models/investment';
import type {AccountInfoResponse} from '@/models/account';
import {purchaseCategory,purchaseDefinitionId} from '@/models/assetPurchase';
import {useAccountsStore} from '@/stores/account';
export function useAssetPurchase(transaction:Ref<Transaction|TransactionTemplate>,enabled:Ref<boolean>){
 const definitionStore=useInvestmentDefinitionsStore();
 const definitions=computed(()=>definitionStore.rows),positions=ref<InvestmentPosition[]>([]),accounts=ref<AccountInfoResponse[]>([]);
 const positionId=ref(''),quantity=ref(''),fee=ref('0'),error=ref(''),busy=ref(false),loading=ref(false),uncertain=ref(false);
 const pending=ref<InvestmentRequest>(),preview=ref<InvestmentDetail>();let sequence=0;
 const definitionId=computed(()=>purchaseDefinitionId(transaction.value.expenseCategoryId||''));
 const active=computed(()=>transaction.value.type===TransactionType.Expense && definitionId.value!==undefined);
 const available=computed(()=>positions.value.filter(p=>p.definitionId===definitionId.value && accounts.value.some(a=>a.id===p.costAccountId && !a.hidden)));
 const position=computed(()=>available.value.find(p=>p.id===positionId.value));
 const category=computed(()=>purchaseCategory(definitions.value));
 const locked=computed(()=>busy.value||!!pending.value);
 const store=useAccountsStore();
 async function load(){const seq=++sequence;if(!enabled.value)return;loading.value=true;error.value='';try{const [,p,a]=await Promise.all([definitionStore.load(),services.getInvestments(),services.getAllAccounts({visibleOnly:true})]);if(seq!==sequence)return;positions.value=p.data.result;accounts.value=a.data.result.flatMap(a=>a.subAccounts?.length?a.subAccounts:[a]);}catch{if(seq===sequence)error.value='投资资产读取失败，请重试。';}finally{if(seq===sequence)loading.value=false;}}
 function applyRecognition(response:RecognizedTransactionResponse){
  if(locked.value)return;
  try{
   const p=position.value;if(!p)throw Error('请先选择资产持仓账户，再使用 AI。');
   const draft=assetRecognitionDraft(response,p.unit,p.unitName||'',p.quantityPrecision??12);
   const t=transaction.value;
   quantity.value=draft.quantity;fee.value=draft.fee;t.sourceAmount=draft.gross;
   const cash=accounts.value.find(a=>a.id===response.sourceAccountId && !a.investmentPositionId && a.currency==='CNY' && !a.hidden);
   t.sourceAccountId=cash?.id || '';
   if(response.time)t.time=response.time;
   if(response.comment)t.comment=response.comment;
   error.value='AI 已预填，请核对数量、成交金额、手续费、付款账户和时间；未能确认的字段已留空。资产及持仓账户保持你的选择，尚未保存。';
  }catch(e){error.value=e instanceof Error?e.message:'识别结果不可用，当前表单未改动。';}
 }
 function reset(){pending.value=undefined;preview.value=undefined;uncertain.value=false;positionId.value='';quantity.value='';fee.value='0';error.value='';}
 watch(definitionId,()=>{if(!pending.value){positionId.value='';quantity.value='';fee.value='0';}});
 watch(available,rows=>{if(rows.length===1&&!positionId.value)positionId.value=rows[0]!.id;});
 watch(enabled,v=>{if(v)void load();},{immediate:true});
 function cancelPreview(){if(uncertain.value){error.value='保存结果尚未确认，请重试原请求，避免重复登记。';return}pending.value=undefined;preview.value=undefined;}
 async function submit():Promise<boolean>{
  if(busy.value)return false;error.value='';busy.value=true;
  try{
   if(!enabled.value||!active.value)throw Error('此入口只用于新增支出里的投资买入。');
   if(!pending.value){
    const p=position.value,t=transaction.value;if(loading.value||!p)throw Error('请选择对应的资产持仓账户。');
    if(!definitions.value.some(d=>d.id===definitionId.value))throw Error('资产定义不可用，请重新选择。');
    const cash=accounts.value.find(a=>a.id===t.sourceAccountId);
    if(!cash||cash.currency!=='CNY'||cash.investmentPositionId||cash.hidden)throw Error('请选择可用的人民币付款账户，不能从持仓账户付款。');
    if(!Number.isSafeInteger(t.sourceAmount)||t.sourceAmount<=0)throw Error('请填写实际成交金额，不含另收手续费。');
    if(t.tagIds.length || ('pictures' in t && t.pictures?.length) || ('geoLocation' in t && t.geoLocation))throw Error('本次资产买入暂不接收标签、位置或图片；请先清除这些内容，凭证可在买入后从资产详情添加。');
    const fresh=(await services.getInvestment(p.id)).data.result.position;
    const req:InvestmentRequest={requestKey:crypto.randomUUID(),positionId:p.id,expectedVersion:fresh.version,operationId:'',kind:'buy',occurredAt:t.time,utcOffset:t.utcOffset,quantity:quantity.value,gross:String(t.sourceAmount),fee:investmentMinorUnits(fee.value),cashAccountId:t.sourceAccountId,transferCategoryId:'0',incomeCategoryId:'0',expenseCategoryId:'0',cancelled:false,comment:t.comment,entryPurpose:'asset_purchase',assetDefinitionId:definitionId.value!};
    preview.value=(await services.previewInvestment(req)).data.result;pending.value=req;return false;
   }
   uncertain.value=true;
   await services.saveInvestment(pending.value);reset();
   try{await store.loadAllAccounts({force:true})}catch{/* Posting succeeded; a later account refresh can recover display. */}
   return true;
  }catch(e){const x=e as {response?:{status?:number;data?:{errorMessage?:string}};message?:string};const status=x.response?.status;if(status&&status>=400&&status<500&&status!==408&&status!==429)uncertain.value=false;error.value=x.response?.data?.errorMessage||x.message||'请求失败，请重试。';return false;}finally{busy.value=false;}
 }
 return reactive({active,available,position,category,definitions,positionId,quantity,fee,error,busy,loading,locked,pending,preview,uncertain,load,applyRecognition,reset,cancelPreview,submit});
}
export type AssetPurchaseForm=ReturnType<typeof useAssetPurchase>;
