import {describe,it,expect} from 'vitest';
import {valuationBalances,type InvestmentValuation} from './investmentValuation';
import type {AccountInfoResponse} from './account';
import {groupTotals} from './accountGroup';
const account={id:'a',investmentPositionId:'p',type:1,category:7,currency:'CNY',balance:2857679} as AccountInfoResponse;
const quote={accountId:'a',positionId:'p',mode:'automatic',status:'reference',cost:'2857679',marketValue:'3063897',validUntil:100} as InvestmentValuation;
describe('reference valuation overlays',()=>{
 it('replaces cost once without mutating ledger values',()=>{const map=valuationBalances([account],[quote],50000);expect(map['a']).toBe(3063897);expect(groupTotals([account],false,{},map)[0]?.assets).toBe(3063897n);expect(account.balance).toBe(2857679)});
 it('rejects stale, failed, wrong owner mapping and changed cost',()=>{expect(valuationBalances([account],[quote],100001)).toEqual({});for(const patch of [{status:'unavailable'},{cost:'5'},{positionId:'foreign'},{marketValue:'NaN'},{marketValue:'99999999999999999'}])expect(valuationBalances([account],[{...quote,...patch}],50000)).toEqual({})});
 it('handles zero value, nested accounts, exclusions and hidden balances',()=>{const parent={id:'parent',type:2,category:7,subAccounts:[account],balance:2857679} as AccountInfoResponse;const map=valuationBalances([parent],[{...quote,marketValue:'0'}],50000);expect(map['a']).toBe(0);expect(groupTotals([parent],false,{},map)[0]?.assets).toBe(0n);expect(groupTotals([parent],false,{a:true},map)).toEqual([]);expect(groupTotals([{...account,hidden:true}],false,{},map)).toEqual([])});
});
