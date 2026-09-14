<template><f7-page><f7-navbar title="机构分组" back-link="返回" /><InstitutionGroups @open-account="props.f7router.navigate($event)" /></f7-page></template>
<script setup lang="ts">import type { Router } from 'framework7/types'; import InstitutionGroups from '@/components/accounts/InstitutionGroups.vue'; const props = defineProps<{ f7router: Router.Router }>();</script>
