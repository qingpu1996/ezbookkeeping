<template>
    <section class="asset-purchase-fields">
        <template v-if="!confirmation">
            <p>填写实际成交金额，手续费另填。买入本金不计入日常消费。</p>
            <label>资产持仓账户
                <select v-model="form.positionId" :disabled="form.locked || form.loading">
                    <option value="">请选择持仓账户</option>
                    <option v-for="p in form.available" :key="p.id" :value="p.id">{{ p.name }}</option>
                </select>
            </label>
            <label>买入数量（{{ form.position?.unitName || '单位' }} / {{ form.position?.unit || '—' }}）
                <input v-model="form.quantity" inputmode="decimal" :disabled="form.locked" placeholder="实际成交数量" />
            </label>
            <label>另收手续费（元）<input v-model="form.fee" inputmode="decimal" :disabled="form.locked" /></label>
            <p v-if="!form.loading && !form.available.length">请先在账户中建立此资产的持仓账户。</p>
        </template>
        <template v-else>
            <p v-if="form.pending && form.preview">
                请确认：买入 {{ form.pending.quantity }} {{ form.position?.unit }}，付款合计
                ¥{{ investmentMoney((BigInt(form.pending.gross) + BigInt(form.pending.fee)).toString()) }}。
                买入后持有 {{ form.preview.calculation.quantity }} {{ form.position?.unit }}，总成本 ¥{{ investmentMoney(form.preview.calculation.cost) }}。
            </p>
            <button v-if="form.pending && !form.uncertain" type="button" :disabled="form.busy" @click="form.cancelPreview()">返回修改</button>
            <p v-if="form.uncertain">正在确认保存结果；重试会使用同一请求，请勿重新录入。</p>
            <p v-if="form.error" role="alert">{{ form.error }}</p>
        </template>
    </section>
</template>
<script setup lang="ts">
import type {AssetPurchaseForm} from '@/composables/useAssetPurchase';
import {investmentMoney} from '@/models/investment';
defineProps<{form:AssetPurchaseForm; confirmation?:boolean}>();
</script>
<style scoped>
.asset-purchase-fields { width:100%; padding: 8px 0; }
.asset-purchase-fields p { margin: 8px 0; line-height:1.6; }
.asset-purchase-fields label { display:block; margin:12px 0; }
.asset-purchase-fields input,.asset-purchase-fields select { display:block; width:100%; padding:12px; border:1px solid #9997; border-radius:6px; color:inherit; background:transparent; font:inherit; }
.asset-purchase-fields button { padding:8px 16px; color:var(--f7-theme-color,#b87945); cursor:pointer; }
[role=alert] { color:#b84225; }
</style>
