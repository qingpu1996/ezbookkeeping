import type {RecognizedTransactionResponse} from './large_language_model';
import {investmentMinorUnits} from './investment';
export function assetRecognitionDraft(response:RecognizedTransactionResponse,unit:string,unitName:string,precision:number){
 const a=response.assetPurchase;
 if(!a || a.kind!=='gold' || a.action!=='buy' || a.currency!=='CNY')throw Error('AI 未确认这是一笔人民币黄金买入，请手动核对填写，当前表单未改动。');
 let quantity='',gross=0,fee='';
 if(typeof a.quantity==='string' && (a.unit===unit || a.unit===unitName) && /^(0|[1-9]\d*)(\.\d{1,12})?$/.test(a.quantity) && (a.quantity.split('.')[1]?.length||0)<=precision && Number(a.quantity)>0)quantity=a.quantity;
 try{if(typeof a.gross==='string')gross=Number(investmentMinorUnits(a.gross));}catch{/* Unknown remains empty. */}
 try{if(typeof a.fee==='string'){investmentMinorUnits(a.fee);fee=a.fee;}}catch{/* Unknown remains empty. */}
 return {quantity,gross,fee};
}
