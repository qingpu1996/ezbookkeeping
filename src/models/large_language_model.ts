export interface RecognizedTransactionResponse {
    readonly assetPurchase?: {kind?:string;action?:string;currency?:string;quantity?:string;unit?:string;gross?:string;fee?:string};
    readonly type: number;
    readonly time?: number;
    readonly categoryId?: string;
    readonly sourceAccountId?: string;
    readonly destinationAccountId?: string;
    readonly sourceAmount?: number;
    readonly destinationAmount?: number;
    readonly tagIds?: string[];
    readonly comment?: string;
}
