// Seed data approximating the kind of curated Chinese dev/tools bookmarks the
// original demo carries (incl. the OneNav "themes" category). categoryId / weight
// drive sidebar + grid ordering; clicks back the popularity counter.

export const seedCategories = [
  { id: 'c-common', name: '常用工具', icon: 'ri:apps-2-line', weight: 0 },
  { id: 'c-dev', name: '开发者', icon: 'ri:code-s-slash-line', weight: 1 },
  { id: 'c-design', name: '设计资源', icon: 'ri:palette-line', weight: 2 },
  { id: 'c-domain', name: '域名 & DNS', icon: 'ri:global-line', weight: 3 },
  { id: 'c-ai', name: 'AI 工具', icon: 'ri:robot-2-line', weight: 4 },
  { id: 'c-media', name: '影视娱乐', icon: 'ri:film-line', weight: 5 },
  { id: 'c-themes', name: 'OneNav 主题', icon: 'ri:layout-masonry-line', weight: 6 },
]

let _id = 0
const L = (categoryId, title, url, description, clicks, extra = {}) => ({
  id: 'l-' + ++_id,
  categoryId,
  title,
  url,
  description,
  clicks,
  weight: 0,
  ...extra,
})

export const seedLinks = [
  // 常用工具
  L('c-common', 'V2EX', 'https://www.v2ex.com', '创意工作者们的社区', 358),
  L('c-common', '小z博客', 'https://xiaoz.me', '记录折腾与分享的个人博客', 96),
  L('c-common', '一纸简历', 'https://www.yzcv.cn', '在线简历制作工具', 41),
  L('c-common', 'WEB 安全色', 'https://www.bootcss.com/p/websafecolors/', 'Web 安全色参考表', 58),

  // 开发者
  L('c-dev', 'Mozilla SSL Configuration Generator', 'https://ssl-config.mozilla.org', '一键生成 Nginx/Apache 的 SSL 配置', 173),
  L('c-dev', 'Bulma', 'https://bulma.io', '基于 Flexbox 的现代 CSS 框架', 219),
  L('c-dev', 'MDN Web Docs', 'https://developer.mozilla.org', 'Web 技术权威文档', 132),
  L('c-dev', 'Can I use', 'https://caniuse.com', '浏览器特性兼容性查询', 88),
  L('c-dev', 'GitHub', 'https://github.com', '全球最大的代码托管平台', 264),

  // 设计资源
  L('c-design', 'Dribbble', 'https://dribbble.com', '设计师作品与灵感社区', 77),
  L('c-design', 'Coolors', 'https://coolors.co', '快速生成配色方案', 64),
  L('c-design', 'unDraw', 'https://undraw.co', '开源可定制的插画素材', 52),
  L('c-design', 'Iconify', 'https://iconify.design', '统一的开源图标框架', 90),

  // 域名 & DNS
  L('c-domain', 'Domain Availability and Price', 'https://tld-list.com', '域名注册价格与可用性查询', 25),
  L('c-domain', '域名历史解析查询', 'https://securitytrails.com', '查询域名的历史解析记录', 47),
  L('c-domain', 'NameSilo', 'https://www.namesilo.com', '便宜稳定的域名注册商', 39),
  L('c-domain', 'DNSPod', 'https://www.dnspod.cn', '免费智能 DNS 解析服务', 61),

  // AI 工具
  L('c-ai', 'ChatGPT', 'https://chat.openai.com', 'OpenAI 对话式 AI', 312),
  L('c-ai', 'ChatGPT 快捷指令', 'https://prompts.chat', '精选 ChatGPT 提示词合集', 358),
  L('c-ai', 'Claude', 'https://claude.ai', 'Anthropic 出品的 AI 助手', 280),
  L('c-ai', 'Midjourney', 'https://www.midjourney.com', 'AI 绘画与图像生成', 201),

  // 影视娱乐
  L('c-media', 'The Movie Database (TMDB)', 'https://www.themoviedb.org', '开放的影视数据库', 254),
  L('c-media', 'IMDb', 'https://www.imdb.com', '全球影视评分与资料库', 143),
  L('c-media', '豆瓣电影', 'https://movie.douban.com', '华语影视评分与影评', 188),

  // OneNav 主题
  L('c-themes', 'webstack - 主题', 'https://github.com/WebStackPage/WebStackPage.github.io', '经典网址导航主题', 333),
  L('c-themes', 'baisuNew - 主题', 'https://github.com/zhuzhuyule/HomeLab-Wiki', '白素清爽导航主题', 330),
  L('c-themes', '5iux - 主题', 'https://github.com/helloxz/onenav', '5iux 风格导航主题', 329),
  L('c-themes', 'sou - 主题', 'https://github.com/helloxz/onenav', '极简搜索式导航主题', 332),
]

// Normalize per-category weights so within-category order is deterministic.
const counters = {}
for (const link of seedLinks) {
  counters[link.categoryId] = counters[link.categoryId] ?? 0
  link.weight = counters[link.categoryId]++
}
