<template><v-card><div class="pa-4"><router-link to="/account/list">返回账户</router-link><router-link v-if="accountId" class="ms-4" :to="'/transaction/list?accountIds=' + encodeURIComponent(accountId)">查看账户流水</router-link></div><GoldLedger :key="accountId" :account-id="accountId" /></v-card></template>
<script setup lang="ts">
import { computed } from 'vue';
import { useRoute } from 'vue-router';
import GoldLedger from '@/components/investments/GoldLedger.vue';
const route = useRoute();
const accountId = computed(() => typeof route.query['accountId'] === 'string' ? route.query['accountId'] : '');
</script>
