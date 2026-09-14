import {describe,it,expect} from 'vitest';
import {purchaseCategory,purchaseDefinitionId} from './assetPurchase';
describe('investment purpose in expense picker',()=>{
 it('uses stable definition identity independently of labels',()=>{
  const d={id:'gold-1',name:'黄金',kind:'gold',unit:'g',unitName:'克',precision:12,version:1,inUse:true};
  const c=purchaseCategory([d]);
  expect(c.name).toBe('投资');
  const id=c.subCategories![0]!.id;
  expect(purchaseDefinitionId(id)).toBe(d.id);
  expect(purchaseCategory([{...d,name:'重新命名'}]).subCategories![0]!.id).toBe(id);
  expect(purchaseDefinitionId('投资')).toBeUndefined();
  expect(purchaseDefinitionId('123456')).toBeUndefined();
 });
});
