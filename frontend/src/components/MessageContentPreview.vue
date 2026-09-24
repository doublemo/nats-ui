<script setup>
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { parseMessage } from '../utils/parseMessage'

const props = defineProps({ message: { type: Object, required: true } })
const { locale } = useI18n()
const tr = (zh, en) => locale.value === 'zh-CN' ? zh : en
const mode = ref('auto')
watch(() => props.message, () => { mode.value = 'auto' })
const result = computed(() => {
  try { return { text: parseMessage(props.message, mode.value), error: '' } }
  catch (err) { return { text: '', error: err.message } }
})
</script>

<template>
  <div class="message-format-viewer">
    <div class="card-header">
      <span class="table-tip">{{ tr('解析方式', 'Parse as') }}</span>
      <el-select v-model="mode" size="small" class="message-format-select">
        <el-option :label="tr('自动识别', 'Auto')" value="auto" />
        <el-option label="String (UTF-8)" value="string" />
        <el-option label="JSON" value="json" />
        <el-option label="Base64 → String" value="base64-string" />
        <el-option label="Base64 → JSON" value="base64-json" />
      </el-select>
    </div>
    <el-alert v-if="result.error" :title="tr('无法按所选格式解析：', 'Cannot parse as selected format: ') + result.error" type="warning" :closable="false" />
    <pre v-else class="payload-view">{{ result.text || tr('空消息', 'Empty message') }}</pre>
    <p v-if="message.truncated" class="table-tip">{{ tr('仅解析前 64 KiB；截断内容可能导致 JSON 无法解析。', 'Only the first 64 KiB is available; truncated JSON may fail to parse.') }}</p>
  </div>
</template>
