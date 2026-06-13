export interface ModelProvider {
  id: string
  name: string
  baseUrl: string
  defaultModel?: string
}

export const MODEL_PROVIDERS: ModelProvider[] = [
  { id: 'deepseek', name: 'DeepSeek', baseUrl: 'https://api.deepseek.com', defaultModel: 'deepseek-chat' },
  { id: 'kimi', name: 'Kimi (Moonshot)', baseUrl: 'https://api.moonshot.cn', defaultModel: 'moonshot-v1-8k' },
  { id: 'qwen', name: 'Qwen (DashScope)', baseUrl: 'https://dashscope.aliyuncs.com/compatible-mode', defaultModel: 'qwen-plus' },
  { id: 'nvidia', name: 'NVIDIA NIM', baseUrl: 'https://integrate.api.nvidia.com/v1', defaultModel: 'meta/llama-3.1-8b-instruct' },
  { id: 'openai', name: 'OpenAI', baseUrl: 'https://api.openai.com/v1', defaultModel: 'gpt-4o' },
  { id: 'siliconflow', name: 'SiliconFlow', baseUrl: 'https://api.siliconflow.cn/v1', defaultModel: 'Qwen/Qwen2.5-7B-Instruct' },
]

export const CUSTOM_PROVIDER_ID = '_custom'

export function findProvider(id: string): ModelProvider | undefined {
  return MODEL_PROVIDERS.find((p) => p.id === id)
}
