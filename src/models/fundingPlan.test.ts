import { describe, it, expect } from 'vitest';
import { fundingSummary, type FundingPlan } from './fundingPlan';
import type { AccountInfoResponse } from './account';
const a = (id: string, balance: number, extra = {}) => ({id, balance, name:id, currency:'CNY', type:1, category:2, ...extra} as AccountInfoResponse);
const plan: FundingPlan = {reserveBasisPoints:0,configured:true,currency:'CNY',reserve:'10000',accountIds:['cash'],version:1};
describe('funding planning',()=>{
 it('excludes protected funds and gold without deducting reserve twice',()=>{
  const r=fundingSummary([a('cash',30000),a('protected',90000),a('gold',2857679,{investmentPositionId:'g',category:7}),a('credit',-50000,{category:3}),a('usd',100000,{currency:'USD'})],plan);
  expect(r).toMatchObject({included:30000n,excluded:90000n,available:20000n,shortfall:0n,ready:true});
 });
 it('handles reserve shortfall and keeps negative included balance',()=>{
  expect(fundingSummary([a('cash',-2000)],plan)).toMatchObject({included:-2000n,available:0n,shortfall:12000n});
 });
 it('requires explicit setup and excludes newly added accounts',()=>{
  expect(fundingSummary([a('cash',30000)],{...plan,configured:false,accountIds:[],reserve:''}).ready).toBe(false);
  expect(fundingSummary([a('cash',30000),a('new',20000)],plan).available).toBe(20000n);
 });
 it('invalidates removed, converted or currency-changed accounts',()=>{
  for(const rows of [[],[a('cash',100,{currency:'USD'})],[a('cash',100,{investmentPositionId:'g'})]])expect(fundingSummary(rows,plan).ready).toBe(false);
 });
 it('uses the higher floor or percentage and rounds reserve upward to cents',()=>{
  const mixed={...plan,reserve:'2000000',reserveBasisPoints:3000};
  expect(fundingSummary([a('cash',5000000)],mixed)).toMatchObject({reserve:2000000n,available:3000000n});
  expect(fundingSummary([a('cash',10000000)],mixed)).toMatchObject({reserve:3000000n,available:7000000n});
  expect(fundingSummary([a('cash',101)],{...plan,reserve:'0',reserveBasisPoints:3333})).toMatchObject({reserve:34n,available:67n});
  expect(fundingSummary([a('cash',101)],{...plan,reserve:'0',reserveBasisPoints:10000}).available).toBe(0n);
 });
 it('counts selected leaves once and preserves large integer sums',()=>{
  const rows=[a('parent',9999999999999,{type:2,subAccounts:[a('cash',9999999999999)]})];
  expect(fundingSummary(rows,plan).included).toBe(9999999999999n);
 });
});
