export interface InvestmentPosition {
    definitionId?: string; unitName?: string; quantityPrecision?: number;
    id: string; name: string; assetType: string; unit: string; currency: string;
    costAccountId: string; quantity: string; cost: string; realized: string;
    algorithm: string; version: number;
}
export interface InvestmentOperation {
    operationId: string; revision: number; kind: 'opening' | 'buy' | 'sell';
    occurredAt: number; utcOffset: number; quantity: string; gross: string; fee: string;
    cashAccountId: string; transferCategoryId: string; incomeCategoryId: string;
    expenseCategoryId: string; cancelled: boolean; comment: string;
}
export interface InvestmentRequest extends Omit<InvestmentOperation, 'revision'> {
    entryPurpose?: 'asset_purchase'; assetDefinitionId?:string;
    requestKey: string; positionId: string; expectedVersion: number;
}
export interface InvestmentCalculation {
    quantity: string; cost: string; realized: string;
    effects: { eventId: string; cashDelta: string; allocatedCost: string; realized: string }[];
}
export interface InvestmentDetail {
    history?: (InvestmentOperation & { id: string; recordedAt: number })[];
    attachments: { pictureId: string; operationId: string; extension: string }[] | null;
    position: InvestmentPosition; operations: InvestmentOperation[]; calculation: InvestmentCalculation;
}
// Currency entry is parsed exactly. No binary floating point multiplication.
export function investmentMinorUnits(value: string): string {
    if (!/^(0|[1-9]\d*)(\.\d{1,2})?$/.test(value)) throw new Error('请输入最多两位小数的非负金额');
    const [whole, fraction = ''] = value.split('.');
    const amount = BigInt(whole!) * 100n + BigInt(fraction.padEnd(2, '0'));
    if (amount > 9999999999999n) throw new Error('金额超出支持范围');
    return amount.toString();
}
export function investmentMoney(minor: string): string {
    const value = BigInt(minor); const absolute = value < 0n ? -value : value;
    return `${value < 0n ? '-' : ''}${absolute / 100n}.${(absolute % 100n).toString().padStart(2, '0')}`;
}

export function investmentAverage(cost: string, quantity: string): string {
    if (!/^(0|[1-9]\d*)(\.\d{1,12})?$/.test(quantity)) return '—';
    const [whole, fraction = ''] = quantity.split('.');
    const q = BigInt(whole!) * 1000000000000n + BigInt(fraction.padEnd(12, '0'));
    if (q === 0n) return '—';
    const scaled = (BigInt(cost) * 100000000000000n + q / 2n) / q;
    return `${scaled / 10000n}.${(scaled % 10000n).toString().padStart(4, '0')}`;
}

export interface InvestmentDefinition { id:string; name:string; kind:string; unit:string; unitName:string; precision:number; version:number; inUse:boolean; }
