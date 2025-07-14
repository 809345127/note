window.onload = function() {
  var code = `st=>start: 用户点击“使用百度账号登录/授权”按钮|past
cv=>operation: 生成 code_verifier、state、code_challenge|past
rd=>operation: 跳转到授权服务器（GET /authorize...）|past
lg=>operation: 展示登录页/授权页|past
au=>condition: 用户登录并同意授权？|past
err1=>end: 跳转回客户端，error=access_denied|past
ok1=>operation: 生成授权码，重定向回客户端，带code|past
tk=>operation: 客户端后端用code换token（POST /token）|past
tkfail=>end: 返回error|past
tkok=>operation: 返回access_token, refresh_token|past
rs=>operation: 客户端用access_token访问资源|past
rsfail=>end: 返回401|past
rsok=>end: 返回资源数据|past

st->cv->rd->lg->au
au(yes)->ok1->tk->tkok->rs->rsok
au(no)->err1
rs->rsfail
rsfail->tkfail
`;
  var diagram = flowchart.parse(code);
  diagram.drawSVG('flowchart', {
    'x': 0,
    'y': 0,
    'line-width': 2,
    'maxWidth': 1000,
    'line-length': 60,
    'text-margin': 10,
    'font-size': 16,
    'font-color': '#34495e',
    'line-color': '#2980b9',
    'element-color': '#2980b9',
    'fill': 'white',
    'yes-text': '是',
    'no-text': '否',
    'arrow-end': 'block',
    'scale': 1,
    'symbols': {
      'start': {
        'font-color': '#fff',
        'element-color': '#27ae60',
        'fill': '#27ae60'
      },
      'end':{
        'font-color': '#fff',
        'element-color': '#c0392b',
        'fill': '#c0392b'
      }
    },
    'flowstate' : {
      'past' : { 'fill' : '#f9f9f9', 'font-size' : 16}
    }
  });
}; 