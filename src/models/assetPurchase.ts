import {TransactionCategory} from '@/models/transaction_category';
import type {InvestmentDefinition} from '@/models/investment';
import {CategoryType} from '@/core/category';
export const ASSET_PURCHASE_PREFIX='asset-purchase:';
export function purchaseDefinitionId(categoryId:string):string|undefined {
 return categoryId.startsWith(ASSET_PURCHASE_PREFIX)?categoryId.slice(ASSET_PURCHASE_PREFIX.length):undefined;
}
export function purchaseCategory(definitions:InvestmentDefinition[]):TransactionCategory {
 const group=TransactionCategory.createNewCategory(CategoryType.Expense);group.id=ASSET_PURCHASE_PREFIX;group.name='投资';group.visible=true;
 group.subCategories=definitions.map(d=>{const item=TransactionCategory.createNewCategory(CategoryType.Expense,group.id);item.id=ASSET_PURCHASE_PREFIX+d.id;item.name=d.name;item.visible=true;return item});return group;
}
