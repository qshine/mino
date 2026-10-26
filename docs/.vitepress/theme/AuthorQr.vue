<script setup lang="ts">
// Renders the author's two QR codes from the book's public assets. A
// component is used so the image paths respect the site base in both
// languages; the surrounding prose stays in the Markdown page.
import { withBase } from 'vitepress'

defineProps<{
  wechat: string
  wechatAlt: string
  xiaohongshu: string
  xiaohongshuAlt: string
}>()
</script>

<template>
  <div class="author-qr">
    <figure>
      <img :src="withBase('/author-wechat-qrcode.jpg')" :alt="wechatAlt" width="168" height="168" loading="lazy" />
      <figcaption>{{ wechat }}</figcaption>
    </figure>
    <figure>
      <img :src="withBase('/author-xiaohongshu-qrcode.jpg')" :alt="xiaohongshuAlt" width="168" height="168" loading="lazy" />
      <figcaption>{{ xiaohongshu }}</figcaption>
    </figure>
  </div>
</template>
