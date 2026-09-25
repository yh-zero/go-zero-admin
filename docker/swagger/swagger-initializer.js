window.onload = async function () {
  try {
    const response = await fetch('./swagger.json', { cache: 'no-store' });
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
    document.title = 'go-zero-admin 接口文档';
  } catch (error) {
    document.getElementById('swagger-ui').textContent =
      `${error.message}。请先生成 Swagger，并检查 Docker 挂载目录。`;
  }
};
