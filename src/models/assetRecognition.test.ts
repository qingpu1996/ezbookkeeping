import {describe,it,expect} from 'vitest';
import {assetRecognitionDraft} from './assetRecognition';
const sample={type:2,assetPurchase:{kind:'gold',action:'buy',currency:'CNY',quantity:'1.2345',unit:'g',gross:'1234.50',fee:'0'}};
describe('asset AI suggestions',()=>{
 it('preserves exact units and distinguishes explicit zero fee',()=>expect(assetRecognitionDraft(sample,'g','克',4)).toEqual({quantity:'1.2345',gross:123450,fee:'0'}));
 it('does not infer quantities, amounts or fees',()=>expect(assetRecognitionDraft({...sample,assetPurchase:{...sample.assetPurchase,unit:'kg',gross:'1e3',fee:''}},'g','克',4)).toEqual({quantity:'',gross:0,fee:''}));
 it('requires a purchase in supported currency',()=>{for(const a of [{action:'sell'},{currency:'USD'},{kind:'silver'}])expect(()=>assetRecognitionDraft({...sample,assetPurchase:{...sample.assetPurchase,...a}},'g','克',4)).toThrow();});
 it('refuses excessive precision',()=>expect(assetRecognitionDraft(sample,'g','克',2).quantity).toBe(''));
});
