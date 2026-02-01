# MySQL 与 ES 请求分配策略

## 当前链路（精确查词）

```
用户查词 → Redis (word→id) → 命中则 MySQL 拉详情 → 未命中则异步 LLM
```

ES 目前只用于：写入（IndexWordDesc）、同步、以及**搜索类**接口（模糊/前缀/中文释义），未参与「精确查词」的读路径。

---

## 推荐分配方式

| 请求类型           | 走谁       | 说明 |
|--------------------|------------|------|
| **精确查词**（已知单词名） | Redis → MySQL（或 Redis → ES → MySQL） | 主数据源建议仍以 MySQL 为准，ES 可作读分流 |
| **模糊/拼写纠错**  | ES         | `SearchWordDescFuzzy` |
| **前缀/联想**      | ES         | `SearchWordDescByWordPrefix` |
| **按中文释义搜**   | ES         | `SearchWordDescByChineseMeaning` |
| **写入/更新**      | MySQL + Redis + ES 都写 | 保持多写一致（你已在做） |

原则：**精确查、强一致走 MySQL（或 Redis+MySQL）；搜索、联想、模糊走 ES。**

---

## 可选：精确查词读分流

若希望减轻 MySQL 压力，可在「Redis 未命中」后：

- **方案 A**：先查 ES（精确 match 单词），命中则用 ES 结果，未命中再查 MySQL，再未命中走 LLM。
- **方案 B**：保持现状，只走 MySQL（ES 仅用于搜索场景）。

实现方案 A 时，ES 需提供「按词精确查」接口（如 `term` 查 `word.keyword`），避免用 `match` 导致分词/不精确。
