<template><v-card><InstitutionGroups @open-account="router.push($event)" /></v-card></template>
<script setup lang="ts">import { useRouter } from 'vue-router'; import InstitutionGroups from '@/components/accounts/InstitutionGroups.vue'; const router = useRouter();</script>
