<template><f7-page><f7-navbar title="账户 · 黄金详情" back-link="返回" /><f7-block><f7-link href="/account/list">返回账户</f7-link><f7-link v-if="accountId" class="margin-left" :href="'/transaction/list?accountIds=' + encodeURIComponent(accountId)">查看账户流水</f7-link></f7-block><GoldLedger :account-id="accountId" /></f7-page></template>
<script setup lang="ts">
import { computed } from 'vue';
import type { Router } from 'framework7/types';
import GoldLedger from '@/components/investments/GoldLedger.vue';
const props = defineProps<{ f7route: Router.Route }>();
const accountId = computed(() => props.f7route.query['accountId'] || '');
</script>
