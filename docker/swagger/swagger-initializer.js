window.onload = async function () {
  try {
    const appAudience = new URLSearchParams(window.location.search).get('audience') === 'app';
    const response = await fetch(appAudience ? './app-swagger.json' : './swagger.json', { cache: 'no-store' });
    if (!response.ok) throw new Error(`接口文档加载失败：HTTP ${response.status}`);
    const spec = await response.json();

    // Only the browser copy uses the proxy; the generated JSON remains portable.
    spec.host = window.location.host;
    spec.schemes = [window.location.protocol.replace(':', '')];

    window.ui = SwaggerUIBundle({
      spec,
      dom_id: '#swagger-ui',
      deepLinking: true,
      filter: true,
      displayRequestDuration: true,
      persistAuthorization: false,
      validatorUrl: null,
      presets: [SwaggerUIBundle.presets.apis],
      layout: 'BaseLayout',
    });
    document.title = appAudience ? 'App 接口文档' : 'go-zero-admin 接口文档';
    const switcher = document.createElement('nav');
    switcher.style.cssText = 'padding:12px 24px;background:#f5f5f5';
    const link = document.createElement('a');
    link.href = appAudience ? '?' : '?audience=app';
    link.textContent = appAudience ? '切换到管理后台接口' : '切换到 App 接口（使用独立 App 登录令牌）';
    switcher.appendChild(link);
    document.body.prepend(switcher);
  } catch (error) {
    document.getElementById('swagger-ui').textContent =
      `${error.message}。请先生成 Swagger，并检查 Docker 挂载目录。`;
  }
};
