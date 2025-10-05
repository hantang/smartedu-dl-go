# 更新记录

## v0.2

【推荐版本】

Tab页面：

- [x] 教材链接输入下载
- [x] 教材查询多选下载
- [x] 课程包查询多选下载
- [x] 语文诵读音频下载

更新：

- `v0.2.0` ~ `v0.2.6`
  - 支持登录信息配置（devtools/network选择pdf文件找到Request Headers中`x-nd-auth`参数）
  - 增加日志统计（结果保存在`log-smartedudl.txt`）
  - 增加备用解析链接
  - 已知问题：
    - 部分音频下载可能失败（包括已配置登录信息）；
    - 非登录状态部分资源可能下载失败或下载的是旧版教材；
    - 新增备用解析，可能导致下载同一个下载多个对应PDF（可能不完全相同）。
- `v0.2.7`
  - 新增课程包Tab页面
  - 支持视频下载（需要登录，单线程，保存格式`.ts`文件，用户可用**FFmpeg**等工具将之转化成其他视频格式）
  - 登录信息可仅配置Access Token
  - 修改字体为“抖音美好体” 来自 [bytedance/fonts][bytedance-fonts]
- `v0.2.8`/`v0.2.9`
  - 修复“精品课程”解析 [#3][issue-3]
- `v0.2.10`
  - 修复保存文件名（去除标题中特殊字符）[#5][issue-5]
- `v0.2.11`
  - 修正保存目录问题 [#4][issue-4] [#6][issue-6]
  - 修复保存文件名（去除标题中特殊字符，含视频资源） [#5][issue-5]
  - 下载资源根据URL路径去除重复
- `v0.2.12`
  - 修复非加密视频下载错误 [#7][issue-7]
  - 支持直接输入资源链接
  - 支持多线程下载视频
- `v0.2.13`
  - 修复同名资源下载错误 [#8][issue-8]
  - 修订提示文字
- `v0.2.14`
  - 修复未选择资源类型时出现下载按钮禁用问题
  - 新增登录信息保存和预加载（或使用环境变量`SMARTEDU_TOKEN`） [#9][issue-9]
  - 新增中小学语文示范诵读库标签页面

[bytedance-fonts]: https://github.com/bytedance/fonts

[issue-3]: https://github.com/hantang/smartedu-dl-go/issues/3
[issue-4]: https://github.com/hantang/smartedu-dl-go/issues/4
[issue-5]: https://github.com/hantang/smartedu-dl-go/issues/5
[issue-6]: https://github.com/hantang/smartedu-dl-go/issues/6
[issue-7]: https://github.com/hantang/smartedu-dl-go/issues/7
[issue-8]: https://github.com/hantang/smartedu-dl-go/issues/8
[issue-9]: https://github.com/hantang/smartedu-dl-go/issues/9

## v0.1

仅支持链接输入列表下载(`v0.1.x`) 【过时】
