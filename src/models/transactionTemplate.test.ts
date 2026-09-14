import { describe, it, expect } from 'vitest';
import { TransactionTemplate, type TransactionTemplateInfoResponse } from './transaction_template';
import { TemplateType } from '@/core/template';
const fixture = (at = 960) => TransactionTemplate.ofTemplate({id:'1',templateType:TemplateType.Schedule.type,name:'test',type:3,categoryId:'1',sourceAccountId:'1',destinationAccountId:'2',sourceAmount:30000,destinationAmount:30000,utcOffset:480,scheduledAt:at,tagIds:[],comment:'',hideAmount:false,editable:true,displayOrder:0,hidden:false,timeSequenceId:'0',time:0,scheduledFrequencyType:3,scheduledFrequency:'1'} as TransactionTemplateInfoResponse);
describe('scheduled local wall time',()=>{
 it('preserves midnight and converts legacy UTC minutes',()=>{expect(fixture().scheduledTime).toBe('00:00');expect(fixture(77).scheduledTime).toBe('09:17')});
 it('round trips both requests and keeps wall time on timezone change',()=>{const t=fixture();t.scheduledTime='09:17';expect(t.toTemplateCreateRequest('test').scheduledMinute).toBe(557);expect(t.toTemplateModifyRequest().scheduledMinute).toBe(557);t.utcOffset=-300;expect(t.scheduledTime).toBe('09:17');const copy=fixture();copy.fillFrom(t);expect(copy.scheduledTime).toBe('09:17')});
 it('ignores invalid or cleared time instead of sending NaN',()=>{const t=fixture(77);for(const value of ['', '24:00','09:60']){t.scheduledTime=value;expect(t.scheduledTime).toBe('09:17')}});
});
