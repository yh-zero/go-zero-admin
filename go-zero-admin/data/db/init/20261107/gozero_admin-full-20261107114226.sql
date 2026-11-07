-- MySQL dump 10.13  Distrib 9.7.4, for Linux (x86_64)
--
-- Host: localhost    Database: gozero_admin
-- ------------------------------------------------------
-- Server version	9.7.4

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!50503 SET NAMES utf8mb4 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;

--
-- Current Database: `gozero_admin`
--

CREATE DATABASE /*!32312 IF NOT EXISTS*/ `gozero_admin` /*!40100 DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci */ /*!80016 DEFAULT ENCRYPTION='N' */;

USE `gozero_admin`;

--
-- Table structure for table `ai_conversation`
--

DROP TABLE IF EXISTS `ai_conversation`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ai_conversation` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `user_id` bigint unsigned NOT NULL COMMENT '所属用户',
  `title` varchar(128) NOT NULL DEFAULT '' COMMENT '会话标题',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  `pinned` tinyint NOT NULL DEFAULT '0' COMMENT '本人置顶',
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`)
) ENGINE=InnoDB AUTO_INCREMENT=8 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 会话';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ai_conversation`
--

LOCK TABLES `ai_conversation` WRITE;
/*!40000 ALTER TABLE `ai_conversation` DISABLE KEYS */;
INSERT INTO `ai_conversation` VALUES (1,1,'这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是1，query用完整问题。仅根据该知识库的检索结','2026-11-06 23:56:09','2026-11-06 23:56:14',NULL,0),(2,1,'这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是1，query用完整问题。仅根据该知识库的检索结','2026-11-06 23:56:14','2026-11-06 23:56:18',NULL,0),(3,1,'这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是2，query用完整问题。仅根据该知识库的检索结','2026-11-06 23:56:18','2026-11-06 23:56:22',NULL,0),(4,1,'这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是2，query用完整问题。仅根据该知识库的检索结','2026-11-06 23:56:22','2026-11-06 23:56:26',NULL,0),(5,1,'这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是1，query用完整问题。仅根据该知识库的检索结','2026-11-06 23:56:26','2026-11-06 23:56:29',NULL,0),(6,1,'这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是2，query用完整问题。仅根据该知识库的检索结','2026-11-06 23:56:29','2026-11-06 23:56:32',NULL,0),(7,1,'UI验收-知识库对话-20261107','2026-11-07 01:41:50','2026-11-07 01:56:04','2026-11-07 01:56:04',1);
/*!40000 ALTER TABLE `ai_conversation` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ai_kb`
--

DROP TABLE IF EXISTS `ai_kb`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ai_kb` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `owner_id` bigint unsigned NOT NULL COMMENT '所属用户（仅本人可读写）',
  `name` varchar(128) NOT NULL COMMENT '知识库名称',
  `description` varchar(512) NOT NULL DEFAULT '' COMMENT '描述',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  KEY `idx_owner` (`owner_id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 知识库';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ai_kb`
--

LOCK TABLES `ai_kb` WRITE;
/*!40000 ALTER TABLE `ai_kb` DISABLE KEYS */;
INSERT INTO `ai_kb` VALUES (1,1,'A股投资知识','演示合成数据；来源 ai_document/ai-rag；KB体验验收20261106；不构成投资建议','2026-11-06 23:55:52','2026-11-06 23:55:52',NULL),(2,1,'运维发布SOP','演示合成数据；来源 ai_document/ai-rag；KB体验验收20261106；不是真实运维规范','2026-11-06 23:55:56','2026-11-06 23:55:56',NULL),(3,1,'UI验收-AI优化-20261107','本轮临时验收，验证后精确删除','2026-11-07 01:36:52','2026-11-07 01:56:04','2026-11-07 01:56:04');
/*!40000 ALTER TABLE `ai_kb` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ai_kb_chunk`
--

DROP TABLE IF EXISTS `ai_kb_chunk`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ai_kb_chunk` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键（即 Qdrant point_id）',
  `doc_id` bigint unsigned NOT NULL COMMENT '文档ID',
  `kb_id` bigint unsigned NOT NULL COMMENT '知识库ID（冗余，检索过滤用）',
  `seq` int unsigned NOT NULL COMMENT '块序号',
  `content` text NOT NULL COMMENT '块正文',
  `token_count` int unsigned NOT NULL DEFAULT '0' COMMENT '估算 token 数',
  `status` varchar(16) NOT NULL DEFAULT 'pending' COMMENT '索引状态：pending/ready/failed',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_doc` (`doc_id`),
  KEY `idx_kb_status` (`kb_id`,`status`)
) ENGINE=InnoDB AUTO_INCREMENT=21 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 知识库分块';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ai_kb_chunk`
--

LOCK TABLES `ai_kb_chunk` WRITE;
/*!40000 ALTER TABLE `ai_kb_chunk` DISABLE KEYS */;
INSERT INTO `ai_kb_chunk` VALUES (1,1,1,0,'# A 股交易基础规则（演示数据）\r\n\r\n> 本文件为知识库联调演示素材，内容为公开常识性规则整理，不构成投资建议。\r\n\r\n## 交易时间\r\n\r\nA 股每个交易日的交易时段为：9:15-9:25 集合竞价（其中 9:20-9:25 不可撤单），9:30-11:30 与 13:00-15:00 连续竞价。14:57-15:00 为深市收盘集合竞价，沪市 2018 年后同样采用收盘集合竞价。法定节假日与周末休市，调休工作日不开市。\r\n\r\n## T+1 制度\r\n\r\nA 股实行 T+1 交收：当日买入的股票次日才能卖出；当日卖出股票所得资金当日可用（可继续买入），次日可取（转出到银行卡）。这与港股、美股的 T+0 有本质区别，短线交易者必须注意当日买入无法当日止损。\r\n\r\n## 委托与成交\r\n\r\n委托方式包括限价委托与市价委托。市价委托有五种类型（最优五档即时成交剩余撤销等）。委托当日有效，未成交部分收盘后自动撤单。买入委托以 100 股（一手）为最小单位，卖出可以是零股；科创板单笔申报最低 200 股，超过 200 股的部分以 1 股递增。\r\n\r\n## 涨跌幅限制\r\n\r\n主板普通股票涨跌幅限',500,'ready','2026-11-06 23:55:52'),(2,1,1,1,'00 股，超过 200 股的部分以 1 股递增。\r\n\r\n## 涨跌幅限制\r\n\r\n主板普通股票涨跌幅限制为 ±10%，ST 股票为 ±5%；创业板与科创板为 ±20%；北交所为 ±30%。新股上市首日规则特殊：主板新股首日最高涨幅 44%，创业板/科创板注册制新股上市前 5 个交易日不设涨跌幅限制，之后恢复 ±20%。',159,'ready','2026-11-06 23:55:52'),(3,2,1,0,'涨跌停与停牌制度（演示数据）\r\n\r\n涨跌停板机制：股票价格达到当日涨跌幅限制时称为涨停或跌停。涨停后仍可卖出（有买单排队），跌停后仍可买入。连续涨跌停常见于有重大利好或利空的股票。\r\n\r\n临时停牌（盘中）：\r\n创业板、科创板股票盘中交易价格较当日开盘价首次上涨或下跌达到或超过 30%、60% 时，实施盘中临时停牌，单次停牌 10 分钟。停牌期间可以申报也可以撤单，复牌时集合竞价。\r\n\r\n长期停牌情形：\r\n1. 重大资产重组：筹划重大事项可申请停牌，交易所对停牌时长有严格约束，原则上不超过 10 个交易日，破产重整等特殊情形除外。\r\n2. 连续异常波动：连续三个交易日内涨跌幅偏离值累计达到 ±20%（主板）属异常波动，需披露异常波动公告，但通常不停牌。\r\n3. 退市整理期：进入退市整理期的股票交易 15 个交易日，主板涨跌幅 ±10%，期间不得筹划重大资产重组。\r\n\r\n新股临时停牌：主板新股上市首日盘中成交价较开盘价首次涨跌达到或超过 10% 时临时停牌 30 分钟，达到或超过 20% 时停牌至 14:55。\r\n\r\n注意：停牌期间持有的股票无法卖出，资金被锁定，这是重仓单票的重要流动性风',500,'ready','2026-11-06 23:55:53'),(4,2,1,1,'停牌至 14:55。\r\n\r\n注意：停牌期间持有的股票无法卖出，资金被锁定，这是重仓单票的重要流动性风险来源。',54,'ready','2026-11-06 23:55:53'),(5,3,1,0,'板块,代码段,涨跌幅限制,开户门槛,最小申报单位,上市制度\r\n沪市主板,600/601/603/605,±10%,无特殊门槛,100股,核准制(注册制过渡完成)\r\n深市主板,000/001/002/003,±10%,无特殊门槛,100股,核准制(注册制过渡完成)\r\n创业板,300/301,±20%,10万元+24个月交易经验,100股,注册制\r\n科创板,688,±20%,50万元+24个月交易经验,200股起按1股递增,注册制\r\n北交所,8开头/920,±30%,50万元+24个月交易经验,100股,注册制\r\nST股票,各板块,*ST为退市风险警示,±5%(主板),无额外门槛,100股,风险警示板',302,'ready','2026-11-06 23:55:54'),(6,4,1,0,'# 融资融券指南（演示数据）\r\n\r\n## 基本概念\r\n\r\n融资 = 向券商借钱买股票（做多杠杆）；融券 = 向券商借股票卖出（做空）。合称两融，账户称为信用账户。开通门槛：50 万元资产 + 半年交易经验，需临柜或线上双录。\r\n\r\n## 关键指标\r\n\r\n- 维持担保比例 =（现金 + 证券市值）/（融资负债 + 融券负债）。警戒线通常为 150%，平仓线 130%。低于平仓线且未按期追加担保物，券商有权强制平仓。\r\n- 融资利率：年化常见 5.0%~8.8%，按日计息（融资天数算头不算尾）。\r\n- 融券费率：通常高于融资利率，且券源有限，热门做空标的经常无券可融。\r\n\r\n## 展期与还款\r\n\r\n单笔融资合约期限最长 6 个月，到期前可申请展期，每次展期不超过 6 个月，展期次数以券商规定为准。还款方式：现金还款或卖券还款（卖出所得优先偿还融资负债）。\r\n\r\n## 风险要点\r\n\r\n两融是杠杆工具：10% 的股价下跌在满仓满融下会放大为约 20% 的本金亏损。强制平仓不保证成交在理想价位，极端行情（连续跌停）下可能出现穿仓——亏损超过本金，投资者仍需补足差额。两融余额也被用作市场情绪指标：',500,'ready','2026-11-06 23:55:55'),(7,4,1,1,'端行情（连续跌停）下可能出现穿仓——亏损超过本金，投资者仍需补足差额。两融余额也被用作市场情绪指标：余额快速上升通常代表杠杆资金看多情绪浓厚。',71,'ready','2026-11-06 23:55:55'),(8,5,1,0,'分红除权与股息红利税（演示数据）\r\n\r\n除权除息：现金分红后股价进行除息调整，除息参考价 = 股权登记日收盘价 - 每股现金红利。送股或转增股本后进行除权，除权参考价 = 股权登记日收盘价 / (1 + 送转比例)。除权除息日当天股票简称会加 XD（除息）、XR（除权）或 DR（除权除息）标记。\r\n\r\n关键日期顺序：预案公告日 → 股东大会通过 → 股权登记日（收盘持有即有权分红）→ 除权除息日 → 红利发放日。在股权登记日收盘时持有股票即可获得分红，次日（除权日）卖出不影响分红资格。\r\n\r\n股息红利税（持股期限差异化，按先进先出计算）：\r\n- 持股 1 个月以内（含）：税负 20%\r\n- 持股 1 个月至 1 年（含）：税负 10%\r\n- 持股超过 1 年：免征\r\n\r\n税款在卖出时由券商代扣代缴。这意味着短线交易者实际到手的分红要打折扣，而除息后股价下调叠加扣税，短期抢权（登记日前买入抢分红）通常没有套利空间。\r\n\r\n送转股对持仓的影响：10 转 10 后持股数量翻倍、股价减半，总市值不变。送转本身不创造价值，但高送转常被市场解读为利好信号（流动性提升、股价绝对值降低）。',494,'ready','2026-11-06 23:55:55'),(9,6,1,0,'{\r\n  \"说明\": \"A 股常见估值与财务指标口径（演示数据）\",\r\n  \"指标\": [\r\n    {\"名称\": \"PE(TTM)\", \"口径\": \"总市值 / 最近四个季度归母净利润之和\", \"备注\": \"亏损企业 PE 为负无意义，应改用 PB 或 PS\"},\r\n    {\"名称\": \"PB(LF)\", \"口径\": \"总市值 / 最新报告期归母净资产\", \"备注\": \"适用于银行、地产等重资产行业\"},\r\n    {\"名称\": \"PS(TTM)\", \"口径\": \"总市值 / 最近四个季度营业收入之和\", \"备注\": \"用于未盈利的成长股\"},\r\n    {\"名称\": \"ROE(加权)\", \"口径\": \"归母净利润 / 加权平均归母净资产\", \"备注\": \"长期 ROE 大于 15% 通常被视为优质公司\"},\r\n    {\"名称\": \"股息率\", \"口径\": \"近 12 个月每股现金分红 / 当前股价\", \"备注\": \"注意区分静态与滚动口径\"},\r\n    {\"名称\": \"换手率\", \"口径\": \"当日成交量 / 流通股本\", \"备注\": \"次新股与热门题材股换手率常超 20%\"',500,'ready','2026-11-06 23:55:56'),(10,6,1,1,', \"口径\": \"当日成交量 / 流通股本\", \"备注\": \"次新股与热门题材股换手率常超 20%\"},\r\n    {\"名称\": \"量比\", \"口径\": \"当日每分钟平均成交量 / 过去 5 日每分钟平均成交量\", \"备注\": \"量比大于 2 视为明显放量\"}\r\n  ]\r\n}',138,'ready','2026-11-06 23:55:56'),(11,7,2,0,'# 生产发布流程规范（演示数据）\r\n\r\n## 发布前\r\n\r\n1. 代码须通过 CI 全部门禁：静态检查、单元测试、构建；禁止带着失败的流水线申请发布。\r\n2. 数据库变更必须先导出备份，回滚 SQL 与变更 SQL 同时评审；破坏性变更（删列、改类型）须先在评审中确认数据处理方案。\r\n3. 发布单须注明：影响范围、回滚步骤、验证清单、值班负责人。缺任何一项打回。\r\n\r\n## 发布窗口\r\n\r\n常规发布窗口为工作日 10:00-17:00。冻结期（大促、节假日保障期）只允许紧急修复发布，须经值班主管与业务负责人双审批。周五 15:00 后原则上不发布非紧急变更。\r\n\r\n## 发布执行\r\n\r\n采用分批灰度：先 1 台验证实例，观察 10 分钟核心指标（错误率、P99 延迟、业务关键日志）无异常后全量。任何一批出现异常立即执行回滚，不允许\"再观察看看\"。\r\n\r\n## 发布后\r\n\r\n发布完成后 30 分钟内值班同学须核对验证清单逐项确认，并在发布单上关闭。48 小时内出现的关联告警须在复盘记录中关联该发布单。',456,'ready','2026-11-06 23:55:56'),(12,8,2,0,'值班与故障响应规范（演示数据）\r\n\r\n值班制度：一线值班 7x24 电话可达，5 分钟内响应告警电话；二线为模块负责人，一线 15 分钟无法定位时升级二线；重大故障 30 分钟内拉齐应急群并指定故障指挥官（Incident Commander），指挥官统一调度，禁止多人各自执行恢复操作。\r\n\r\n故障分级：\r\nP0 全站不可用或核心交易中断：5 分钟内响应，30 分钟内给出处置方案，恢复后 24 小时内出复盘报告。\r\nP1 核心功能受损但有降级方案：15 分钟响应，2 小时内恢复或降级。\r\nP2 非核心功能异常：工作时间 4 小时内处理。\r\nP3 体验类问题：排期修复。\r\n\r\n故障处置原则：先恢复、再定位、后追责。恢复手段优先级：回滚 > 降级 > 重启 > 修复上线。任何人在故障期间禁止直接修改生产数据库数据，确需修数须故障指挥官批准并双人复核。\r\n\r\n复盘要求：复盘只对流程不对人（blameless）；必须产出根因、时间线、改进项与负责人、完成期限。改进项逾期未闭环升级至部门负责人。',449,'ready','2026-11-06 23:55:57'),(13,9,2,0,'变更类型,窗口要求,回滚时限,审批链,备注\r\n常规应用发布,工作日10:00-17:00,15分钟内完成,模块负责人,灰度先行\r\n数据库结构变更,工作日10:00-12:00,依赖备份恢复时长,DBA+模块负责人,回滚SQL随变更评审\r\n中间件配置变更,低峰期(13:00-15:00),10分钟内,中间件负责人,先改单机验证\r\n网络/防火墙变更,周二/周四下午,5分钟内回退,网络组+安全组,变更期间双人复核\r\n紧急修复,任意时间,立即,值班主管+业务负责人,事后24小时补评审\r\n大促冻结期变更,禁止,不适用,例外需部门负责人,只允许P0修复',273,'ready','2026-11-06 23:55:57'),(14,10,2,0,'{\r\n  \"说明\": \"监控告警分级与通知渠道（演示数据）\",\r\n  \"级别\": [\r\n    {\"级别\": \"critical\", \"定义\": \"服务不可用或核心指标越限\", \"通知\": \"电话+短信+IM\", \"响应\": \"5分钟\", \"示例\": \"全站5xx错误率>5%持续2分钟\"},\r\n    {\"级别\": \"warning\", \"定义\": \"指标异常但未影响用户\", \"通知\": \"IM群\", \"响应\": \"30分钟\", \"示例\": \"单机CPU>85%持续10分钟\"},\r\n    {\"级别\": \"info\", \"定义\": \"需要关注但不需立即处理\", \"通知\": \"日报汇总\", \"响应\": \"次工作日\", \"示例\": \"磁盘使用率>70%\"}\r\n  ],\r\n  \"静默规则\": \"同一告警30分钟内不重复电话通知；计划内变更期间可临时静默，须登记静默窗口与范围\",\r\n  \"升级规则\": \"critical告警10分钟未认领自动升级至值班主管，30分钟未恢复升级至部门负责人\"\r\n}',450,'ready','2026-11-06 23:55:57'),(15,11,2,0,'# 数据库变更规范（演示数据）\r\n\r\n## 变更分类\r\n\r\n- 低风险：加索引、加可空列、扩大 varchar 长度。工作日窗口内执行，DBA 复核即可。\r\n- 中风险：新建表、删除索引、修改列默认值。须评审 + 备份 + 回滚方案。\r\n- 高风险：删列、改列类型、删表、大表 DDL。原则上禁止直接执行，须走\"建新表 + 双写 + 迁移 + 校验 + 切换\"的绕行方案；大表使用在线 DDL 工具并限速。\r\n\r\n## 执行纪律\r\n\r\n1. 任何变更前先确认目标实例与库名，禁止凭连接别名记忆操作；变更脚本与回滚脚本成对评审。\r\n2. 完整备份优先：变更前导出受影响表的快照或全库备份，记录导出时间与校验值；禁止用初始化脚本代替运行库备份。\r\n3. 禁止全表无条件 UPDATE/DELETE；批量数据修复须分批、带 WHERE 边界并先 SELECT 核对行数。\r\n4. 变更完成后 30 分钟内观察慢查询与错误日志，确认无回归后关闭变更单。\r\n\r\n## 缓存一致性\r\n\r\n涉及被缓存行的变更，须在提交后精确失效对应缓存键；禁止用 FLUSHALL/FLUSHDB 代替精确失效。缓存失效失败时必须',500,'ready','2026-11-06 23:55:58'),(16,11,2,1,'提交后精确失效对应缓存键；禁止用 FLUSHALL/FLUSHDB 代替精确失效。缓存失效失败时必须在变更单中记录\"库已提交、缓存未清\"的状态，不得宣称已回滚。',80,'ready','2026-11-06 23:55:58');
/*!40000 ALTER TABLE `ai_kb_chunk` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ai_kb_doc`
--

DROP TABLE IF EXISTS `ai_kb_doc`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ai_kb_doc` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `kb_id` bigint unsigned NOT NULL COMMENT '知识库ID',
  `title` varchar(128) NOT NULL COMMENT '文档标题',
  `content` mediumtext NOT NULL COMMENT '原文（一期内联文本，≤64KiB）',
  `status` varchar(16) NOT NULL DEFAULT 'pending' COMMENT '摄取状态：pending/ready/failed',
  `error` varchar(512) NOT NULL DEFAULT '' COMMENT '安全错误信息',
  `chunk_count` int unsigned NOT NULL DEFAULT '0' COMMENT '分块数',
  `embedding_model` varchar(128) NOT NULL DEFAULT '' COMMENT '使用的 embedding 模型版本',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  `version` bigint unsigned NOT NULL DEFAULT '1' COMMENT '正文修订号，替换加一',
  `ingest_stage` varchar(16) NOT NULL DEFAULT 'embedding' COMMENT 'embedding/indexing/ready/failed',
  PRIMARY KEY (`id`),
  KEY `idx_kb` (`kb_id`)
) ENGINE=InnoDB AUTO_INCREMENT=15 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 知识库文档';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ai_kb_doc`
--

LOCK TABLES `ai_kb_doc` WRITE;
/*!40000 ALTER TABLE `ai_kb_doc` DISABLE KEYS */;
INSERT INTO `ai_kb_doc` VALUES (1,1,'交易基础规则','# A 股交易基础规则（演示数据）\r\n\r\n> 本文件为知识库联调演示素材，内容为公开常识性规则整理，不构成投资建议。\r\n\r\n## 交易时间\r\n\r\nA 股每个交易日的交易时段为：9:15-9:25 集合竞价（其中 9:20-9:25 不可撤单），9:30-11:30 与 13:00-15:00 连续竞价。14:57-15:00 为深市收盘集合竞价，沪市 2018 年后同样采用收盘集合竞价。法定节假日与周末休市，调休工作日不开市。\r\n\r\n## T+1 制度\r\n\r\nA 股实行 T+1 交收：当日买入的股票次日才能卖出；当日卖出股票所得资金当日可用（可继续买入），次日可取（转出到银行卡）。这与港股、美股的 T+0 有本质区别，短线交易者必须注意当日买入无法当日止损。\r\n\r\n## 委托与成交\r\n\r\n委托方式包括限价委托与市价委托。市价委托有五种类型（最优五档即时成交剩余撤销等）。委托当日有效，未成交部分收盘后自动撤单。买入委托以 100 股（一手）为最小单位，卖出可以是零股；科创板单笔申报最低 200 股，超过 200 股的部分以 1 股递增。\r\n\r\n## 涨跌幅限制\r\n\r\n主板普通股票涨跌幅限制为 ±10%，ST 股票为 ±5%；创业板与科创板为 ±20%；北交所为 ±30%。新股上市首日规则特殊：主板新股首日最高涨幅 44%，创业板/科创板注册制新股上市前 5 个交易日不设涨跌幅限制，之后恢复 ±20%。\r\n','ready','',2,'text-embedding-v4','2026-11-06 23:55:52','2026-11-07 01:22:09',NULL,1,'ready'),(2,1,'涨跌停与停牌制度','涨跌停与停牌制度（演示数据）\r\n\r\n涨跌停板机制：股票价格达到当日涨跌幅限制时称为涨停或跌停。涨停后仍可卖出（有买单排队），跌停后仍可买入。连续涨跌停常见于有重大利好或利空的股票。\r\n\r\n临时停牌（盘中）：\r\n创业板、科创板股票盘中交易价格较当日开盘价首次上涨或下跌达到或超过 30%、60% 时，实施盘中临时停牌，单次停牌 10 分钟。停牌期间可以申报也可以撤单，复牌时集合竞价。\r\n\r\n长期停牌情形：\r\n1. 重大资产重组：筹划重大事项可申请停牌，交易所对停牌时长有严格约束，原则上不超过 10 个交易日，破产重整等特殊情形除外。\r\n2. 连续异常波动：连续三个交易日内涨跌幅偏离值累计达到 ±20%（主板）属异常波动，需披露异常波动公告，但通常不停牌。\r\n3. 退市整理期：进入退市整理期的股票交易 15 个交易日，主板涨跌幅 ±10%，期间不得筹划重大资产重组。\r\n\r\n新股临时停牌：主板新股上市首日盘中成交价较开盘价首次涨跌达到或超过 10% 时临时停牌 30 分钟，达到或超过 20% 时停牌至 14:55。\r\n\r\n注意：停牌期间持有的股票无法卖出，资金被锁定，这是重仓单票的重要流动性风险来源。\r\n','ready','',2,'text-embedding-v4','2026-11-06 23:55:53','2026-11-07 01:22:09',NULL,1,'ready'),(3,1,'板块差异对照','板块,代码段,涨跌幅限制,开户门槛,最小申报单位,上市制度\r\n沪市主板,600/601/603/605,±10%,无特殊门槛,100股,核准制(注册制过渡完成)\r\n深市主板,000/001/002/003,±10%,无特殊门槛,100股,核准制(注册制过渡完成)\r\n创业板,300/301,±20%,10万元+24个月交易经验,100股,注册制\r\n科创板,688,±20%,50万元+24个月交易经验,200股起按1股递增,注册制\r\n北交所,8开头/920,±30%,50万元+24个月交易经验,100股,注册制\r\nST股票,各板块,*ST为退市风险警示,±5%(主板),无额外门槛,100股,风险警示板\r\n','ready','',1,'text-embedding-v4','2026-11-06 23:55:54','2026-11-07 01:22:09',NULL,1,'ready'),(4,1,'融资融券指南','# 融资融券指南（演示数据）\r\n\r\n## 基本概念\r\n\r\n融资 = 向券商借钱买股票（做多杠杆）；融券 = 向券商借股票卖出（做空）。合称两融，账户称为信用账户。开通门槛：50 万元资产 + 半年交易经验，需临柜或线上双录。\r\n\r\n## 关键指标\r\n\r\n- 维持担保比例 =（现金 + 证券市值）/（融资负债 + 融券负债）。警戒线通常为 150%，平仓线 130%。低于平仓线且未按期追加担保物，券商有权强制平仓。\r\n- 融资利率：年化常见 5.0%~8.8%，按日计息（融资天数算头不算尾）。\r\n- 融券费率：通常高于融资利率，且券源有限，热门做空标的经常无券可融。\r\n\r\n## 展期与还款\r\n\r\n单笔融资合约期限最长 6 个月，到期前可申请展期，每次展期不超过 6 个月，展期次数以券商规定为准。还款方式：现金还款或卖券还款（卖出所得优先偿还融资负债）。\r\n\r\n## 风险要点\r\n\r\n两融是杠杆工具：10% 的股价下跌在满仓满融下会放大为约 20% 的本金亏损。强制平仓不保证成交在理想价位，极端行情（连续跌停）下可能出现穿仓——亏损超过本金，投资者仍需补足差额。两融余额也被用作市场情绪指标：余额快速上升通常代表杠杆资金看多情绪浓厚。\r\n','ready','',2,'text-embedding-v4','2026-11-06 23:55:55','2026-11-07 01:22:09',NULL,1,'ready'),(5,1,'分红除权与股息税','分红除权与股息红利税（演示数据）\r\n\r\n除权除息：现金分红后股价进行除息调整，除息参考价 = 股权登记日收盘价 - 每股现金红利。送股或转增股本后进行除权，除权参考价 = 股权登记日收盘价 / (1 + 送转比例)。除权除息日当天股票简称会加 XD（除息）、XR（除权）或 DR（除权除息）标记。\r\n\r\n关键日期顺序：预案公告日 → 股东大会通过 → 股权登记日（收盘持有即有权分红）→ 除权除息日 → 红利发放日。在股权登记日收盘时持有股票即可获得分红，次日（除权日）卖出不影响分红资格。\r\n\r\n股息红利税（持股期限差异化，按先进先出计算）：\r\n- 持股 1 个月以内（含）：税负 20%\r\n- 持股 1 个月至 1 年（含）：税负 10%\r\n- 持股超过 1 年：免征\r\n\r\n税款在卖出时由券商代扣代缴。这意味着短线交易者实际到手的分红要打折扣，而除息后股价下调叠加扣税，短期抢权（登记日前买入抢分红）通常没有套利空间。\r\n\r\n送转股对持仓的影响：10 转 10 后持股数量翻倍、股价减半，总市值不变。送转本身不创造价值，但高送转常被市场解读为利好信号（流动性提升、股价绝对值降低）。\r\n','ready','',1,'text-embedding-v4','2026-11-06 23:55:55','2026-11-07 01:22:09',NULL,1,'ready'),(6,1,'常见指标口径','{\r\n  \"说明\": \"A 股常见估值与财务指标口径（演示数据）\",\r\n  \"指标\": [\r\n    {\"名称\": \"PE(TTM)\", \"口径\": \"总市值 / 最近四个季度归母净利润之和\", \"备注\": \"亏损企业 PE 为负无意义，应改用 PB 或 PS\"},\r\n    {\"名称\": \"PB(LF)\", \"口径\": \"总市值 / 最新报告期归母净资产\", \"备注\": \"适用于银行、地产等重资产行业\"},\r\n    {\"名称\": \"PS(TTM)\", \"口径\": \"总市值 / 最近四个季度营业收入之和\", \"备注\": \"用于未盈利的成长股\"},\r\n    {\"名称\": \"ROE(加权)\", \"口径\": \"归母净利润 / 加权平均归母净资产\", \"备注\": \"长期 ROE 大于 15% 通常被视为优质公司\"},\r\n    {\"名称\": \"股息率\", \"口径\": \"近 12 个月每股现金分红 / 当前股价\", \"备注\": \"注意区分静态与滚动口径\"},\r\n    {\"名称\": \"换手率\", \"口径\": \"当日成交量 / 流通股本\", \"备注\": \"次新股与热门题材股换手率常超 20%\"},\r\n    {\"名称\": \"量比\", \"口径\": \"当日每分钟平均成交量 / 过去 5 日每分钟平均成交量\", \"备注\": \"量比大于 2 视为明显放量\"}\r\n  ]\r\n}\r\n','ready','',2,'text-embedding-v4','2026-11-06 23:55:56','2026-11-07 01:22:09',NULL,1,'ready'),(7,2,'发布流程规范','# 生产发布流程规范（演示数据）\r\n\r\n## 发布前\r\n\r\n1. 代码须通过 CI 全部门禁：静态检查、单元测试、构建；禁止带着失败的流水线申请发布。\r\n2. 数据库变更必须先导出备份，回滚 SQL 与变更 SQL 同时评审；破坏性变更（删列、改类型）须先在评审中确认数据处理方案。\r\n3. 发布单须注明：影响范围、回滚步骤、验证清单、值班负责人。缺任何一项打回。\r\n\r\n## 发布窗口\r\n\r\n常规发布窗口为工作日 10:00-17:00。冻结期（大促、节假日保障期）只允许紧急修复发布，须经值班主管与业务负责人双审批。周五 15:00 后原则上不发布非紧急变更。\r\n\r\n## 发布执行\r\n\r\n采用分批灰度：先 1 台验证实例，观察 10 分钟核心指标（错误率、P99 延迟、业务关键日志）无异常后全量。任何一批出现异常立即执行回滚，不允许\"再观察看看\"。\r\n\r\n## 发布后\r\n\r\n发布完成后 30 分钟内值班同学须核对验证清单逐项确认，并在发布单上关闭。48 小时内出现的关联告警须在复盘记录中关联该发布单。\r\n','ready','',1,'text-embedding-v4','2026-11-06 23:55:56','2026-11-07 01:22:09',NULL,1,'ready'),(8,2,'值班与故障响应','值班与故障响应规范（演示数据）\r\n\r\n值班制度：一线值班 7x24 电话可达，5 分钟内响应告警电话；二线为模块负责人，一线 15 分钟无法定位时升级二线；重大故障 30 分钟内拉齐应急群并指定故障指挥官（Incident Commander），指挥官统一调度，禁止多人各自执行恢复操作。\r\n\r\n故障分级：\r\nP0 全站不可用或核心交易中断：5 分钟内响应，30 分钟内给出处置方案，恢复后 24 小时内出复盘报告。\r\nP1 核心功能受损但有降级方案：15 分钟响应，2 小时内恢复或降级。\r\nP2 非核心功能异常：工作时间 4 小时内处理。\r\nP3 体验类问题：排期修复。\r\n\r\n故障处置原则：先恢复、再定位、后追责。恢复手段优先级：回滚 > 降级 > 重启 > 修复上线。任何人在故障期间禁止直接修改生产数据库数据，确需修数须故障指挥官批准并双人复核。\r\n\r\n复盘要求：复盘只对流程不对人（blameless）；必须产出根因、时间线、改进项与负责人、完成期限。改进项逾期未闭环升级至部门负责人。\r\n','ready','',1,'text-embedding-v4','2026-11-06 23:55:57','2026-11-07 01:22:09',NULL,1,'ready'),(9,2,'变更窗口与回滚','变更类型,窗口要求,回滚时限,审批链,备注\r\n常规应用发布,工作日10:00-17:00,15分钟内完成,模块负责人,灰度先行\r\n数据库结构变更,工作日10:00-12:00,依赖备份恢复时长,DBA+模块负责人,回滚SQL随变更评审\r\n中间件配置变更,低峰期(13:00-15:00),10分钟内,中间件负责人,先改单机验证\r\n网络/防火墙变更,周二/周四下午,5分钟内回退,网络组+安全组,变更期间双人复核\r\n紧急修复,任意时间,立即,值班主管+业务负责人,事后24小时补评审\r\n大促冻结期变更,禁止,不适用,例外需部门负责人,只允许P0修复\r\n','ready','',1,'text-embedding-v4','2026-11-06 23:55:57','2026-11-07 01:22:09',NULL,1,'ready'),(10,2,'监控告警分级','{\r\n  \"说明\": \"监控告警分级与通知渠道（演示数据）\",\r\n  \"级别\": [\r\n    {\"级别\": \"critical\", \"定义\": \"服务不可用或核心指标越限\", \"通知\": \"电话+短信+IM\", \"响应\": \"5分钟\", \"示例\": \"全站5xx错误率>5%持续2分钟\"},\r\n    {\"级别\": \"warning\", \"定义\": \"指标异常但未影响用户\", \"通知\": \"IM群\", \"响应\": \"30分钟\", \"示例\": \"单机CPU>85%持续10分钟\"},\r\n    {\"级别\": \"info\", \"定义\": \"需要关注但不需立即处理\", \"通知\": \"日报汇总\", \"响应\": \"次工作日\", \"示例\": \"磁盘使用率>70%\"}\r\n  ],\r\n  \"静默规则\": \"同一告警30分钟内不重复电话通知；计划内变更期间可临时静默，须登记静默窗口与范围\",\r\n  \"升级规则\": \"critical告警10分钟未认领自动升级至值班主管，30分钟未恢复升级至部门负责人\"\r\n}\r\n','ready','',1,'text-embedding-v4','2026-11-06 23:55:57','2026-11-07 01:22:09',NULL,1,'ready'),(11,2,'数据库变更规范','# 数据库变更规范（演示数据）\r\n\r\n## 变更分类\r\n\r\n- 低风险：加索引、加可空列、扩大 varchar 长度。工作日窗口内执行，DBA 复核即可。\r\n- 中风险：新建表、删除索引、修改列默认值。须评审 + 备份 + 回滚方案。\r\n- 高风险：删列、改列类型、删表、大表 DDL。原则上禁止直接执行，须走\"建新表 + 双写 + 迁移 + 校验 + 切换\"的绕行方案；大表使用在线 DDL 工具并限速。\r\n\r\n## 执行纪律\r\n\r\n1. 任何变更前先确认目标实例与库名，禁止凭连接别名记忆操作；变更脚本与回滚脚本成对评审。\r\n2. 完整备份优先：变更前导出受影响表的快照或全库备份，记录导出时间与校验值；禁止用初始化脚本代替运行库备份。\r\n3. 禁止全表无条件 UPDATE/DELETE；批量数据修复须分批、带 WHERE 边界并先 SELECT 核对行数。\r\n4. 变更完成后 30 分钟内观察慢查询与错误日志，确认无回归后关闭变更单。\r\n\r\n## 缓存一致性\r\n\r\n涉及被缓存行的变更，须在提交后精确失效对应缓存键；禁止用 FLUSHALL/FLUSHDB 代替精确失效。缓存失效失败时必须在变更单中记录\"库已提交、缓存未清\"的状态，不得宣称已回滚。\r\n','ready','',2,'text-embedding-v4','2026-11-06 23:55:58','2026-11-07 01:22:09',NULL,1,'ready'),(12,2,'KB验收临时探针-20261106-真实提交','知识库真实提交验收：探针口令是蓝松果-20261106。此文档仅用于创建、重复提交和删除验证。','ready','',1,'text-embedding-v4','2026-11-06 23:59:13','2026-11-07 01:22:09','2026-11-07 00:00:18',1,'ready'),(13,3,'验收资料甲','临时验收资料甲更新版：工单提交后五分钟内确认。\n保留原始换行与中文。','ready','',1,'text-embedding-v4','2026-11-07 01:37:37','2026-11-07 01:56:04','2026-11-07 01:56:04',2,'ready'),(14,3,'验收资料乙','# 临时验收资料乙\n\n发布前需要检查备份、回滚方案和变更窗口。异常处理结果应记录在工单中。\n','ready','',1,'text-embedding-v4','2026-11-07 01:37:37','2026-11-07 01:56:04','2026-11-07 01:56:04',1,'ready');
/*!40000 ALTER TABLE `ai_kb_doc` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ai_message`
--

DROP TABLE IF EXISTS `ai_message`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ai_message` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `conversation_id` bigint unsigned NOT NULL COMMENT '会话ID',
  `role` varchar(16) NOT NULL COMMENT '角色：user/assistant/tool/system',
  `content` text NOT NULL COMMENT '消息内容',
  `tool_calls` json DEFAULT NULL COMMENT '调用元数据，兼容历史工具调用数组',
  `tokens` int unsigned NOT NULL DEFAULT '0' COMMENT '消耗 token 数',
  `status` varchar(16) NOT NULL DEFAULT 'succeeded' COMMENT '调用状态：running/succeeded/failed',
  `call_id` varchar(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL COMMENT '用户命名空间内的幂等调用标识，历史消息为空',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_call_role` (`call_id`,`role`),
  KEY `idx_conversation` (`conversation_id`)
) ENGINE=InnoDB AUTO_INCREMENT=15 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 消息';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ai_message`
--

LOCK TABLES `ai_message` WRITE;
/*!40000 ALTER TABLE `ai_message` DISABLE KEYS */;
INSERT INTO `ai_message` VALUES (1,1,'user','这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是1，query用完整问题。仅根据该知识库的检索结果回答，并注明文档标题；禁止使用其他知识库或模型常识补足资料。未找到明确依据就说指定知识库资料不足，不得猜测。问题：根据本演示资料，当天买入的A股下午跌了3%，可以当天卖出止损吗？卖出资金何时可用、可取？','{\"calls\": [{\"id\": \"call_00_UfTWtWRSX9YdFy3lEOT70651\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"kb_id\\\": 1, \\\"query\\\": \\\"当天买入的A股下午跌了3%，可以当天卖出止损吗？卖出资金何时可用、可取？\\\"}\"}}], \"version\": 1, \"fingerprint\": \"170f65bf66587d58b463730b4d170d120a4ae74bfc0dc3e3da179a7d1f7adc88\", \"usage_complete\": true}',0,'succeeded','1:e9301b88-5cf2-430c-acc6-7a0b83d3a9e3','2026-11-06 23:56:09'),(2,1,'assistant','**结论：不能当天卖出止损。**\n\n依据知识库 doc_id=1《交易基础规则》中“T+1 制度”一节：\n\n- A 股实行 **T+1 交收：当日买入的股票次日才能卖出**。因此，当天上午买入的 A 股即使下午跌了 3%，也**无法在当天卖出止损**；该文档还明确写有“短线交易者必须注意当日买入无法当日止损”。\n- **卖出资金的到账与可用时间**：当日卖出股票所得资金**当日可用**（可以继续买入），**次日可取**（转出到银行卡）。\n\n也就是说，“T+1”限制的是**股票**的卖出时点（次日才能卖），而资金是“当日可用、次日可取”。本次检索到的其他片段（doc_id=2《涨跌停与停牌制度》、doc_id=4《融资融券指南》）没有针对该问题的规则，因此不能作为依据。\n\n*说明：以上仅依据知识库 1（kb_id=1）的检索片段作答；片段为不完整摘录，内容为演示数据，不构成投资建议。*','{\"calls\": [{\"id\": \"call_00_UfTWtWRSX9YdFy3lEOT70651\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"kb_id\\\": 1, \\\"query\\\": \\\"当天买入的A股下午跌了3%，可以当天卖出止损吗？卖出资金何时可用、可取？\\\"}\"}}], \"version\": 1, \"fingerprint\": \"170f65bf66587d58b463730b4d170d120a4ae74bfc0dc3e3da179a7d1f7adc88\", \"usage_complete\": true}',2974,'succeeded','1:e9301b88-5cf2-430c-acc6-7a0b83d3a9e3','2026-11-06 23:56:09'),(3,2,'user','这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是1，query用完整问题。仅根据该知识库的检索结果回答，并注明文档标题；禁止使用其他知识库或模型常识补足资料。未找到明确依据就说指定知识库资料不足，不得猜测。问题：根据本演示资料，持股8个月卖出，股息红利税税率是多少？登记日收盘持有、除权日卖出是否仍有分红资格？','{\"calls\": [{\"id\": \"call_00_tq843rYpDldLAl7EbC9x4308\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"query\\\": \\\"持股8个月卖出，股息红利税税率是多少？登记日收盘持有、除权日卖出是否仍有分红资格？\\\", \\\"kb_id\\\": 1}\"}}], \"version\": 1, \"fingerprint\": \"4763eff379d2b0ed14135b1a54cec170805c0a0bc2c991dd536ab1ef421cd318\", \"usage_complete\": true}',0,'succeeded','1:5685678f-d5f2-4024-b39b-5a7749c168be','2026-11-06 23:56:14'),(4,2,'assistant','以下是**仅依据知识库 kb_id=1 的检索结果**作出的回答，依据文档标题：《分红除权与股息税》（演示数据）。\n\n**1. 持股 8 个月卖出的股息红利税税率：10%**\n该文档列出的股息红利税按持股期限差异化征收（按先进先出计算）：\n- 持股 1 个月以内（含）：税负 20%\n- 持股 1 个月至 1 年（含）：税负 10%\n- 持股超过 1 年：免征\n\n持股 8 个月属于“1 个月至 1 年（含）”这一档，因此税负为 **10%**。税款在卖出时由券商代扣代缴。\n\n**2. 登记日收盘持有、除权日卖出：仍有分红资格**\n该文档明确写明：\n- 关键日期顺序：预案公告日 → 股东大会通过 → 股权登记日（收盘持有即有权分红）→ 除权除息日 → 红利发放日；\n- “在股权登记日收盘时持有股票即可获得分红，**次日（除权日）卖出不影响分红资格**”。\n\n因此，除权日卖出只是会使分红被扣税，不会影响取得分红的资格；并且除息参考价＝股权登记日收盘价 − 每股现金红利，除权日当天股票简称会加上 XD、XR 或 DR 标记。\n\n注意：以上内容仅是演示资料片段的摘录，具体适用时请以正式规则原文为准；本知识库中未涉及的其他细节不作补充。','{\"calls\": [{\"id\": \"call_00_tq843rYpDldLAl7EbC9x4308\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"query\\\": \\\"持股8个月卖出，股息红利税税率是多少？登记日收盘持有、除权日卖出是否仍有分红资格？\\\", \\\"kb_id\\\": 1}\"}}], \"version\": 1, \"fingerprint\": \"4763eff379d2b0ed14135b1a54cec170805c0a0bc2c991dd536ab1ef421cd318\", \"usage_complete\": true}',3100,'succeeded','1:5685678f-d5f2-4024-b39b-5a7749c168be','2026-11-06 23:56:14'),(5,3,'user','这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是2，query用完整问题。仅根据该知识库的检索结果回答，并注明文档标题；禁止使用其他知识库或模型常识补足资料。未找到明确依据就说指定知识库资料不足，不得猜测。问题：根据本演示资料，critical告警10分钟未认领与30分钟未恢复分别升级给谁？','{\"calls\": [{\"id\": \"call_00_WKFVrGgMXcaiNAQDaaM27350\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"query\\\": \\\"根据本演示资料，critical告警10分钟未认领与30分钟未恢复分别升级给谁？\\\", \\\"kb_id\\\": 2}\"}}], \"version\": 1, \"fingerprint\": \"0a9eae97d3b900a1de9abf9cfb66300608bf84d47c37ee5792904a27012812b8\", \"usage_complete\": true}',0,'succeeded','1:028ef6be-83e1-4a00-830b-222021cdc881','2026-11-06 23:56:18'),(6,3,'assistant','根据知识库（kb_id=2）的检索结果，答案如下：\n\n- **critical 告警 10 分钟未认领**：自动升级至**值班主管**。\n- **critical 告警 30 分钟未恢复**：升级至**部门负责人**。\n\n依据文档：**《监控告警分级》**（kb_id=2，doc_id=10），其“升级规则”原文为：“critical告警10分钟未认领自动升级至值班主管，30分钟未恢复升级至部门负责人”。\n\n（该资料为演示数据，以上仅基于指定知识库检索片段作答，未使用其他知识库或常识补足。）','{\"calls\": [{\"id\": \"call_00_WKFVrGgMXcaiNAQDaaM27350\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"query\\\": \\\"根据本演示资料，critical告警10分钟未认领与30分钟未恢复分别升级给谁？\\\", \\\"kb_id\\\": 2}\"}}], \"version\": 1, \"fingerprint\": \"0a9eae97d3b900a1de9abf9cfb66300608bf84d47c37ee5792904a27012812b8\", \"usage_complete\": true}',2822,'succeeded','1:028ef6be-83e1-4a00-830b-222021cdc881','2026-11-06 23:56:18'),(7,4,'user','这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是2，query用完整问题。仅根据该知识库的检索结果回答，并注明文档标题；禁止使用其他知识库或模型常识补足资料。未找到明确依据就说指定知识库资料不足，不得猜测。问题：根据本演示资料，数据库变更已提交但缓存失效失败，变更单应该如何记录？可以写已回滚或用FLUSHALL处理吗？','{\"calls\": [{\"id\": \"call_00_Qcrcs4MlstlK9751IRoz8709\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"query\\\": \\\"数据库变更已提交但缓存失效失败，变更单应该如何记录？可以写已回滚或用FLUSHALL处理吗？\\\", \\\"kb_id\\\": 2}\"}}], \"version\": 1, \"fingerprint\": \"cb5d1f88390171558155bde949d6c5c758287f205af63b18459ecbdb6f17912e\", \"usage_complete\": true}',0,'succeeded','1:20b84c26-b457-4363-ac65-b1222f6ecc57','2026-11-06 23:56:22'),(8,4,'assistant','**依据文档：《数据库变更规范》**（kb_id=2，doc_id=11）\n\n## 结论\n\n**不可以**写“已回滚”，也**不可以**用 FLUSHALL/FLUSHDB 代替精确失效。\n\n## 规范原文依据\n\n1. **记录要求**：规范“缓存一致性”一节明确要求——缓存失效失败时必须在变更单中记录 **“库已提交、缓存未清”** 的状态，**不得宣称已回滚**。也就是说，变更单必须如实反映实际情况：数据库侧已经提交，缓存侧未清理；不能把它记成回滚成功。\n\n2. **FLUSHALL 禁止**：同一节明确“涉及被缓存行的变更，须在提交后精确失效对应缓存键；**禁止用 FLUSHALL/FLUSHDB 代替精确失效**”。因此，不能把清空整个缓存库当作清掉这次变更对应缓存键的替代做法。\n\n## 资料边界说明\n\n- 关于“缓存失效失败后具体如何补偿、如何重试、如何告警”的操作步骤，本次检索到的片段中**没有明确条款**（片段在“缓存失效失败时必须”后即截断）。按验收要求，我不作推测补足；如需该细节，请提供该知识库更完整的《数据库变更规范》内容。\n- 其他文档（《变更窗口与回滚》《发布流程规范》《值班与故障响应》）只涉及变更窗口、回滚 SQL 评审、恢复优先级等，**未对本题的记录方式作出规定**，故未作为作答依据。','{\"calls\": [{\"id\": \"call_00_Qcrcs4MlstlK9751IRoz8709\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"query\\\": \\\"数据库变更已提交但缓存失效失败，变更单应该如何记录？可以写已回滚或用FLUSHALL处理吗？\\\", \\\"kb_id\\\": 2}\"}}], \"version\": 1, \"fingerprint\": \"cb5d1f88390171558155bde949d6c5c758287f205af63b18459ecbdb6f17912e\", \"usage_complete\": true}',3120,'succeeded','1:20b84c26-b457-4363-ac65-b1222f6ecc57','2026-11-06 23:56:22'),(9,5,'user','这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是1，query用完整问题。仅根据该知识库的检索结果回答，并注明文档标题；禁止使用其他知识库或模型常识补足资料。未找到明确依据就说指定知识库资料不足，不得猜测。问题：本演示知识库规定周五常规发布的截止时间是什么？仅查指定投资知识库，资料没有就明确说没有。','{\"calls\": [{\"id\": \"call_00_uB0u94YpZzKJdZaUaxrL7304\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"kb_id\\\": 1, \\\"query\\\": \\\"本演示知识库规定周五常规发布的截止时间是什么？\\\"}\"}}], \"version\": 1, \"fingerprint\": \"3ca8c89b4b717a8909bfc7af0cc63b8b726641b4f1516ce8ff42bd56c57460f2\", \"usage_complete\": true}',0,'succeeded','1:69489c55-6350-4671-8f6b-901c0949736c','2026-11-06 23:56:26'),(10,5,'assistant','**指定知识库（kb_id=1）资料不足，无法回答。**\n\n在该知识库的检索结果中，没有找到任何关于“周五常规发布截止时间”的内容。检索到的文档为《交易基础规则》《涨跌停与停牌制度》《融资融券指南》《常见指标口径》，内容涉及A股交易时间、涨跌停与停牌、两融规则、指标口径等，均未规定发布截止时间。\n\n以上仅为基于该指定知识库检索结果的判断，未使用其他知识库或模型常识补足。','{\"calls\": [{\"id\": \"call_00_uB0u94YpZzKJdZaUaxrL7304\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"kb_id\\\": 1, \\\"query\\\": \\\"本演示知识库规定周五常规发布的截止时间是什么？\\\"}\"}}], \"version\": 1, \"fingerprint\": \"3ca8c89b4b717a8909bfc7af0cc63b8b726641b4f1516ce8ff42bd56c57460f2\", \"usage_complete\": true}',2864,'succeeded','1:69489c55-6350-4671-8f6b-901c0949736c','2026-11-06 23:56:26'),(11,6,'user','这是合成演示资料的验收问题。请务必先调用kb_search，参数kb_id必须是2，query用完整问题。仅根据该知识库的检索结果回答，并注明文档标题；禁止使用其他知识库或模型常识补足资料。未找到明确依据就说指定知识库资料不足，不得猜测。问题：本演示知识库规定A股持股8个月的股息税税率是多少？仅查指定运维知识库，资料没有就明确说没有。','{\"calls\": [{\"id\": \"call_00_io3h9TqOoiCLVnrE348u3673\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"kb_id\\\": 2, \\\"query\\\": \\\"本演示知识库规定A股持股8个月的股息税税率是多少？\\\"}\"}}], \"version\": 1, \"fingerprint\": \"21d0e3ed6c249fb82355b211b9b480f732b51ff081957edb8f11065d078845ca\", \"usage_complete\": true}',0,'succeeded','1:a713d370-3e92-4942-893d-56185e8dde67','2026-11-06 23:56:29'),(12,6,'assistant','**结论：指定知识库（kb_id=2）资料不足，未找到\"A股持股8个月股息税税率\"的任何依据。**\n\n检索情况说明：\n\n- 已按要求调用 `kb_search`，参数 `kb_id=2`，query 为完整问题。\n- 返回的命中片段全部为运维类文档，与本问题无关：\n  - 《数据库变更规范》（doc_id 11，得分 0.326/0.322）\n  - 《值班与故障响应》（doc_id 8，0.253）\n  - 《监控告警分级》（doc_id 10，0.252）\n  - 《发布流程规范》（doc_id 7，0.247）\n- 上述文档涉及数据库变更分类与纪律、值班与故障分级、告警通知渠道、发布流程与灰度回滚，均未提及A股、股息税、持股期限或任何税率数值。\n\n因此，按照\"仅根据指定知识库、不得猜测、不得用模型常识补足\"的要求，本演示知识库没有该问题的答案。若需要，请确认是否上传了正确的知识库，或提供包含税务规定的资料后我再检索。','{\"calls\": [{\"id\": \"call_00_io3h9TqOoiCLVnrE348u3673\", \"type\": \"function\", \"index\": 0, \"function\": {\"name\": \"kb_search\", \"arguments\": \"{\\\"kb_id\\\": 2, \\\"query\\\": \\\"本演示知识库规定A股持股8个月的股息税税率是多少？\\\"}\"}}], \"version\": 1, \"fingerprint\": \"21d0e3ed6c249fb82355b211b9b480f732b51ff081957edb8f11065d078845ca\", \"usage_complete\": true}',3101,'succeeded','1:a713d370-3e92-4942-893d-56185e8dde67','2026-11-06 23:56:29'),(13,7,'user','数据库变更提交成功但缓存清理失败时，应该如何记录和补偿？','{\"calls\": [], \"sources\": [{\"kb_id\": 2, \"score\": 0.7567095, \"doc_id\": 11, \"snippet\": \"提交后精确失效对应缓存键；禁止用 FLUSHALL/FLUSHDB 代替精确失效。缓存失效失败时必须在变更单中记录\\\"库已提交、缓存未清\\\"的状态，不得宣称已回滚。\", \"chunk_id\": 16, \"doc_title\": \"数据库变更规范\"}, {\"kb_id\": 2, \"score\": 0.70409054, \"doc_id\": 11, \"snippet\": \"# 数据库变更规范（演示数据）\\r\\n\\r\\n## 变更分类\\r\\n\\r\\n- 低风险：加索引、加可空列、扩大 varchar 长度。工作日窗口内执行，DBA 复核即可。\\r\\n- 中风险：新建表、删除索引、修改列默认值。须评审 + 备份 + 回滚方案。\\r\\n- 高风险：删列、改列类型、删表、大表 DDL。原则上禁止直接执行，须走\\\"建新表 + 双写 + 迁移 + 校验 + 切换\\\"的绕行方案；大表使用在线 DDL 工具并限速。\\r\\n\\r\\n## 执行纪律\\r\\n\\r\\n1. 任何变更前先确认目标实例与库名，禁止凭连接别名记忆操作；变更脚本与回滚脚本成对评审。\\r\\n2. 完整备份优先：变更前导出受影响表的快照或全库备份，记录导出时间与校验值；禁止用初始化脚本代替运行库备份。\\r\\n3. 禁止全表无条件 UPDATE/DELETE；批量数据修复须分批、带 WHERE 边界并先 SELECT 核对行数。\\r\\n4. 变更完成后 30 分钟内观察慢查询与错误日志，确认无回归后关闭变更单。\\r\\n\\r\\n## 缓存一致性\\r\\n\\r\\n涉及被缓存行的变更，须在提交后精确失效对应缓存键；禁止用 FLUSHALL/FLUSHDB 代替精确失效。缓存失效失败时必须\", \"chunk_id\": 15, \"doc_title\": \"数据库变更规范\"}, {\"kb_id\": 2, \"score\": 0.5417811, \"doc_id\": 7, \"snippet\": \"# 生产发布流程规范（演示数据）\\r\\n\\r\\n## 发布前\\r\\n\\r\\n1. 代码须通过 CI 全部门禁：静态检查、单元测试、构建；禁止带着失败的流水线申请发布。\\r\\n2. 数据库变更必须先导出备份，回滚 SQL 与变更 SQL 同时评审；破坏性变更（删列、改类型）须先在评审中确认数据处理方案。\\r\\n3. 发布单须注明：影响范围、回滚步骤、验证清单、值班负责人。缺任何一项打回。\\r\\n\\r\\n## 发布窗口\\r\\n\\r\\n常规发布窗口为工作日 10:00-17:00。冻结期（大促、节假日保障期）只允许紧急修复发布，须经值班主管与业务负责人双审批。周五 15:00 后原则上不发布非紧急变更。\\r\\n\\r\\n## 发布执行\\r\\n\\r\\n采用分批灰度：先 1 台验证实例，观察 10 分钟核心指标（错误率、P99 延迟、业务关键日志）无异常后全量。任何一批出现异常立即执行回滚，不允许\\\"再观察看看\\\"。\\r\\n\\r\\n## 发布后\\r\\n\\r\\n发布完成后 30 分钟内值班同学须核对验证清单逐项确认，并在发布单上关闭。48 小时内出现的关联告警须在复盘记录中关联该发布单。\", \"chunk_id\": 11, \"doc_title\": \"发布流程规范\"}, {\"kb_id\": 2, \"score\": 0.53960425, \"doc_id\": 9, \"snippet\": \"变更类型,窗口要求,回滚时限,审批链,备注\\r\\n常规应用发布,工作日10:00-17:00,15分钟内完成,模块负责人,灰度先行\\r\\n数据库结构变更,工作日10:00-12:00,依赖备份恢复时长,DBA+模块负责人,回滚SQL随变更评审\\r\\n中间件配置变更,低峰期(13:00-15:00),10分钟内,中间件负责人,先改单机验证\\r\\n网络/防火墙变更,周二/周四下午,5分钟内回退,网络组+安全组,变更期间双人复核\\r\\n紧急修复,任意时间,立即,值班主管+业务负责人,事后24小时补评审\\r\\n大促冻结期变更,禁止,不适用,例外需部门负责人,只允许P0修复\", \"chunk_id\": 13, \"doc_title\": \"变更窗口与回滚\"}, {\"kb_id\": 2, \"score\": 0.49795574, \"doc_id\": 8, \"snippet\": \"值班与故障响应规范（演示数据）\\r\\n\\r\\n值班制度：一线值班 7x24 电话可达，5 分钟内响应告警电话；二线为模块负责人，一线 15 分钟无法定位时升级二线；重大故障 30 分钟内拉齐应急群并指定故障指挥官（Incident Commander），指挥官统一调度，禁止多人各自执行恢复操作。\\r\\n\\r\\n故障分级：\\r\\nP0 全站不可用或核心交易中断：5 分钟内响应，30 分钟内给出处置方案，恢复后 24 小时内出复盘报告。\\r\\nP1 核心功能受损但有降级方案：15 分钟响应，2 小时内恢复或降级。\\r\\nP2 非核心功能异常：工作时间 4 小时内处理。\\r\\nP3 体验类问题：排期修复。\\r\\n\\r\\n故障处置原则：先恢复、再定位、后追责。恢复手段优先级：回滚 > 降级 > 重启 > 修复上线。任何人在故障期间禁止直接修改生产数据库数据，确需修数须故障指挥官批准并双人复核。\\r\\n\\r\\n复盘要求：复盘只对流程不对人（blameless）；必须产出根因、时间线、改进项与负责人、完成期限。改进项逾期未闭环升级至部门负责人。\", \"chunk_id\": 12, \"doc_title\": \"值班与故障响应\"}], \"version\": 1, \"fingerprint\": \"6fdc67add7dde1c5afd37970de9900f41b165038414bde1e1634bacc24a4ae39\", \"usage_complete\": true}',0,'succeeded','1:a2149190-a33d-4c65-9ce0-5e266d1c7bce','2026-11-07 01:41:50'),(14,7,'assistant','**关于记录：** 涉及被缓存行的变更，须在提交后精确失效对应缓存键，禁止用 FLUSHALL/FLUSHDB 代替精确失效；缓存失效失败时，必须在变更单中记录“库已提交、缓存未清”的状态，且不得宣称已回滚 [1][2]。\n\n**关于补偿：** 现有资料摘录仅明确了上述记录与如实声明的要求，未提供缓存清理失败后的具体补偿操作（如重试、手工清键、对账修复等）的规范内容，依据不足，无法给出补偿步骤。','{\"calls\": [], \"sources\": [{\"kb_id\": 2, \"score\": 0.7567095, \"doc_id\": 11, \"snippet\": \"提交后精确失效对应缓存键；禁止用 FLUSHALL/FLUSHDB 代替精确失效。缓存失效失败时必须在变更单中记录\\\"库已提交、缓存未清\\\"的状态，不得宣称已回滚。\", \"chunk_id\": 16, \"doc_title\": \"数据库变更规范\"}, {\"kb_id\": 2, \"score\": 0.70409054, \"doc_id\": 11, \"snippet\": \"# 数据库变更规范（演示数据）\\r\\n\\r\\n## 变更分类\\r\\n\\r\\n- 低风险：加索引、加可空列、扩大 varchar 长度。工作日窗口内执行，DBA 复核即可。\\r\\n- 中风险：新建表、删除索引、修改列默认值。须评审 + 备份 + 回滚方案。\\r\\n- 高风险：删列、改列类型、删表、大表 DDL。原则上禁止直接执行，须走\\\"建新表 + 双写 + 迁移 + 校验 + 切换\\\"的绕行方案；大表使用在线 DDL 工具并限速。\\r\\n\\r\\n## 执行纪律\\r\\n\\r\\n1. 任何变更前先确认目标实例与库名，禁止凭连接别名记忆操作；变更脚本与回滚脚本成对评审。\\r\\n2. 完整备份优先：变更前导出受影响表的快照或全库备份，记录导出时间与校验值；禁止用初始化脚本代替运行库备份。\\r\\n3. 禁止全表无条件 UPDATE/DELETE；批量数据修复须分批、带 WHERE 边界并先 SELECT 核对行数。\\r\\n4. 变更完成后 30 分钟内观察慢查询与错误日志，确认无回归后关闭变更单。\\r\\n\\r\\n## 缓存一致性\\r\\n\\r\\n涉及被缓存行的变更，须在提交后精确失效对应缓存键；禁止用 FLUSHALL/FLUSHDB 代替精确失效。缓存失效失败时必须\", \"chunk_id\": 15, \"doc_title\": \"数据库变更规范\"}, {\"kb_id\": 2, \"score\": 0.5417811, \"doc_id\": 7, \"snippet\": \"# 生产发布流程规范（演示数据）\\r\\n\\r\\n## 发布前\\r\\n\\r\\n1. 代码须通过 CI 全部门禁：静态检查、单元测试、构建；禁止带着失败的流水线申请发布。\\r\\n2. 数据库变更必须先导出备份，回滚 SQL 与变更 SQL 同时评审；破坏性变更（删列、改类型）须先在评审中确认数据处理方案。\\r\\n3. 发布单须注明：影响范围、回滚步骤、验证清单、值班负责人。缺任何一项打回。\\r\\n\\r\\n## 发布窗口\\r\\n\\r\\n常规发布窗口为工作日 10:00-17:00。冻结期（大促、节假日保障期）只允许紧急修复发布，须经值班主管与业务负责人双审批。周五 15:00 后原则上不发布非紧急变更。\\r\\n\\r\\n## 发布执行\\r\\n\\r\\n采用分批灰度：先 1 台验证实例，观察 10 分钟核心指标（错误率、P99 延迟、业务关键日志）无异常后全量。任何一批出现异常立即执行回滚，不允许\\\"再观察看看\\\"。\\r\\n\\r\\n## 发布后\\r\\n\\r\\n发布完成后 30 分钟内值班同学须核对验证清单逐项确认，并在发布单上关闭。48 小时内出现的关联告警须在复盘记录中关联该发布单。\", \"chunk_id\": 11, \"doc_title\": \"发布流程规范\"}, {\"kb_id\": 2, \"score\": 0.53960425, \"doc_id\": 9, \"snippet\": \"变更类型,窗口要求,回滚时限,审批链,备注\\r\\n常规应用发布,工作日10:00-17:00,15分钟内完成,模块负责人,灰度先行\\r\\n数据库结构变更,工作日10:00-12:00,依赖备份恢复时长,DBA+模块负责人,回滚SQL随变更评审\\r\\n中间件配置变更,低峰期(13:00-15:00),10分钟内,中间件负责人,先改单机验证\\r\\n网络/防火墙变更,周二/周四下午,5分钟内回退,网络组+安全组,变更期间双人复核\\r\\n紧急修复,任意时间,立即,值班主管+业务负责人,事后24小时补评审\\r\\n大促冻结期变更,禁止,不适用,例外需部门负责人,只允许P0修复\", \"chunk_id\": 13, \"doc_title\": \"变更窗口与回滚\"}, {\"kb_id\": 2, \"score\": 0.49795574, \"doc_id\": 8, \"snippet\": \"值班与故障响应规范（演示数据）\\r\\n\\r\\n值班制度：一线值班 7x24 电话可达，5 分钟内响应告警电话；二线为模块负责人，一线 15 分钟无法定位时升级二线；重大故障 30 分钟内拉齐应急群并指定故障指挥官（Incident Commander），指挥官统一调度，禁止多人各自执行恢复操作。\\r\\n\\r\\n故障分级：\\r\\nP0 全站不可用或核心交易中断：5 分钟内响应，30 分钟内给出处置方案，恢复后 24 小时内出复盘报告。\\r\\nP1 核心功能受损但有降级方案：15 分钟响应，2 小时内恢复或降级。\\r\\nP2 非核心功能异常：工作时间 4 小时内处理。\\r\\nP3 体验类问题：排期修复。\\r\\n\\r\\n故障处置原则：先恢复、再定位、后追责。恢复手段优先级：回滚 > 降级 > 重启 > 修复上线。任何人在故障期间禁止直接修改生产数据库数据，确需修数须故障指挥官批准并双人复核。\\r\\n\\r\\n复盘要求：复盘只对流程不对人（blameless）；必须产出根因、时间线、改进项与负责人、完成期限。改进项逾期未闭环升级至部门负责人。\", \"chunk_id\": 12, \"doc_title\": \"值班与故障响应\"}], \"version\": 1, \"feedback\": \"helpful\", \"fingerprint\": \"6fdc67add7dde1c5afd37970de9900f41b165038414bde1e1634bacc24a4ae39\", \"feedback_note\": \"来源能够支持该结论；本轮界面验收。\", \"usage_complete\": true}',1581,'succeeded','1:a2149190-a33d-4c65-9ce0-5e266d1c7bce','2026-11-07 01:41:50');
/*!40000 ALTER TABLE `ai_message` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ai_task`
--

DROP TABLE IF EXISTS `ai_task`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ai_task` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `workflow_id` bigint unsigned DEFAULT NULL COMMENT '工作流ID（对话触发的任务为空）',
  `conversation_id` bigint unsigned DEFAULT NULL COMMENT '会话ID',
  `status` varchar(24) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/running/pending_approval/succeeded/failed/rejected',
  `input` text COMMENT '输入（JSON）',
  `result` text COMMENT '执行结果',
  `error` varchar(512) NOT NULL DEFAULT '' COMMENT '错误信息',
  `approved_by` bigint unsigned DEFAULT NULL COMMENT '审批人ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `completed_at` datetime DEFAULT NULL COMMENT '完成时间',
  `created_by` bigint unsigned NOT NULL DEFAULT '0' COMMENT '任务所属用户，0表示历史归属待确认',
  `next_step` int unsigned NOT NULL DEFAULT '0' COMMENT '下一待执行步骤索引',
  `workflow_snapshot` json DEFAULT NULL COMMENT '执行时工作流定义快照',
  `request_id` varchar(36) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL COMMENT '用户内运行幂等ID',
  `step_records` json DEFAULT NULL COMMENT '已解析步骤输入及执行记录',
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_owner_request` (`created_by`,`request_id`),
  KEY `idx_workflow` (`workflow_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 任务';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ai_task`
--

LOCK TABLES `ai_task` WRITE;
/*!40000 ALTER TABLE `ai_task` DISABLE KEYS */;
/*!40000 ALTER TABLE `ai_task` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `ai_workflow`
--

DROP TABLE IF EXISTS `ai_workflow`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `ai_workflow` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `name` varchar(128) NOT NULL COMMENT '工作流名称',
  `definition` json NOT NULL COMMENT 'JSON DSL：{"steps":[{"name","tool","input","approval"}]}',
  `interval_seconds` int unsigned NOT NULL DEFAULT '0' COMMENT '调度间隔秒数，0=仅手动触发',
  `enabled` tinyint NOT NULL DEFAULT '1' COMMENT '是否启用',
  `next_run_at` datetime DEFAULT NULL COMMENT '下次执行时间',
  `created_by` bigint unsigned NOT NULL COMMENT '创建者用户ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  KEY `idx_next_run` (`enabled`,`next_run_at`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='AI 工作流';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `ai_workflow`
--

LOCK TABLES `ai_workflow` WRITE;
/*!40000 ALTER TABLE `ai_workflow` DISABLE KEYS */;
INSERT INTO `ai_workflow` VALUES (1,'UI验收-知识库分析-20261107-已编辑','{\"steps\": [{\"name\": \"知识库分析\", \"tool\": \"kb_analyze\", \"input\": {\"kb_id\": 2, \"query\": \"根据数据库变更规范，说明数据库已经提交但缓存失效失败时应该如何记录，能否宣称回滚或执行FLUSHALL，并给出简短操作清单。\"}, \"approval\": true}]}',0,0,NULL,1,'2026-11-07 00:56:38','2026-11-07 01:05:58','2026-11-07 01:05:58'),(2,'UI验收-参数传递-20261107','{\"steps\": [{\"id\": \"15a3b3bf-f8b7-480e-b3e9-074e6a9d280d\", \"name\": \"整理本次 JSON\", \"tool\": \"json_formatter\", \"input\": {\"text\": {\"$ref\": \"input.text\"}}, \"approval\": true}, {\"id\": \"5812afd6-29d3-4479-bb49-9ec64cf60a04\", \"name\": \"统计整理结果\", \"tool\": \"text_stats\", \"input\": {\"text\": {\"$ref\": \"steps.15a3b3bf-f8b7-480e-b3e9-074e6a9d280d\"}}, \"approval\": false}], \"version\": 2}',0,0,NULL,1,'2026-11-07 01:39:51','2026-11-07 01:56:04','2026-11-07 01:56:04'),(3,'知识库分析与复核','{\"steps\": [{\"id\": \"6ab52715-1027-40f4-86c4-f42884135d3a\", \"name\": \"分析知识库资料\", \"tool\": \"kb_analyze\", \"input\": {\"kb_id\": 2, \"query\": {\"$ref\": \"input.question\"}}, \"approval\": false}, {\"id\": \"eed5481f-7b0b-42fd-9409-adfbf2b8058f\", \"name\": \"统计分析文本\", \"tool\": \"text_stats\", \"input\": {\"text\": {\"$ref\": \"steps.6ab52715-1027-40f4-86c4-f42884135d3a.analysis\"}}, \"approval\": false}], \"version\": 2}',0,1,NULL,1,'2026-11-07 01:41:04','2026-11-07 01:41:04',NULL);
/*!40000 ALTER TABLE `ai_workflow` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_feedback`
--

DROP TABLE IF EXISTS `app_feedback`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_feedback` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `user_id` bigint unsigned NOT NULL COMMENT '提交用户',
  `content` text NOT NULL COMMENT '纯文字反馈内容',
  `reply` text NOT NULL COMMENT '后台回复，空字符串为未回复',
  `status` varchar(16) NOT NULL DEFAULT 'pending' COMMENT 'pending/resolved',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user_id` (`user_id`,`id`),
  KEY `idx_status_id` (`status`,`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='App 意见反馈';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_feedback`
--

LOCK TABLES `app_feedback` WRITE;
/*!40000 ALTER TABLE `app_feedback` DISABLE KEYS */;
INSERT INTO `app_feedback` VALUES (1,1,'建议在账户设置中增加操作指引，方便新用户快速开始。','','pending','2026-01-01 00:00:00','2026-01-01 00:00:00'),(2,2,'会员状态说明很清晰，希望补充有效期的展示示例。','感谢建议，会员页面已展示生效时间和到期时间。','resolved','2026-01-01 00:00:00','2026-01-01 00:00:00'),(3,1,'这是一条用于演示处理流程的虚构反馈。','','pending','2026-01-01 00:00:00','2026-01-01 00:00:00');
/*!40000 ALTER TABLE `app_feedback` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_membership_grant`
--

DROP TABLE IF EXISTS `app_membership_grant`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_membership_grant` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `user_id` bigint unsigned NOT NULL COMMENT '用户ID',
  `plan_code` varchar(32) NOT NULL COMMENT '方案编码（常量：vip）',
  `benefits` varchar(512) NOT NULL DEFAULT '' COMMENT '权益快照（逗号分隔，如 profile.export）',
  `request_id` varchar(64) NOT NULL DEFAULT '' COMMENT '幂等请求ID（续期重试复用）',
  `granted_by` bigint unsigned DEFAULT NULL COMMENT '操作管理员ID（自助开通为 NULL）',
  `starts_at` datetime NOT NULL COMMENT '生效时间',
  `expires_at` datetime NOT NULL COMMENT '过期时间',
  `revoked` tinyint NOT NULL DEFAULT '0' COMMENT '是否已撤销',
  `revoked_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_request` (`request_id`),
  KEY `idx_user` (`user_id`),
  KEY `idx_expires` (`expires_at`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='App 会员授予';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_membership_grant`
--

LOCK TABLES `app_membership_grant` WRITE;
/*!40000 ALTER TABLE `app_membership_grant` DISABLE KEYS */;
INSERT INTO `app_membership_grant` VALUES (1,1,'pro','profile.export','demo-membership-active',1,'2026-10-30 14:45:25','2026-12-06 14:45:25',0,NULL,'2026-01-01 00:00:00','2026-01-01 00:00:00'),(2,2,'basic','','demo-membership-expired',1,'2026-08-08 14:45:25','2026-10-07 14:45:25',0,NULL,'2026-01-01 00:00:00','2026-01-01 00:00:00'),(3,2,'pro','profile.export','demo-membership-revoked',1,'2026-10-27 14:45:25','2026-11-26 14:45:25',1,'2026-11-06 14:45:25','2026-01-01 00:00:00','2026-01-01 00:00:00'),(4,2,'basic','','demo-membership-upcoming',1,'2026-11-13 14:45:25','2026-12-13 14:45:25',0,NULL,'2026-01-01 00:00:00','2026-01-01 00:00:00');
/*!40000 ALTER TABLE `app_membership_grant` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `app_user`
--

DROP TABLE IF EXISTS `app_user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `app_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `account` varchar(128) NOT NULL COMMENT '账号（邮箱或手机号，小写）',
  `identity_kind` varchar(16) NOT NULL COMMENT '账号类型：email/phone',
  `password_hash` varchar(128) NOT NULL COMMENT '密码哈希（bcrypt）',
  `nickname` varchar(64) NOT NULL DEFAULT '' COMMENT '昵称',
  `remark` varchar(500) NOT NULL DEFAULT '' COMMENT '仅后台可见的内部备注',
  `avatar_file_id` bigint unsigned DEFAULT NULL COMMENT '头像文件ID',
  `status` varchar(16) NOT NULL DEFAULT 'pending' COMMENT '状态：pending/active/disabled/rejected',
  `review_reason` varchar(512) NOT NULL DEFAULT '' COMMENT '审核备注',
  `email_verified_at` datetime DEFAULT NULL COMMENT '邮箱验证时间',
  `phone_verified_at` datetime DEFAULT NULL COMMENT '手机验证时间',
  `last_login_at` datetime DEFAULT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_account` (`account`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=6 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='App 用户';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `app_user`
--

LOCK TABLES `app_user` WRITE;
/*!40000 ALTER TABLE `app_user` DISABLE KEYS */;
INSERT INTO `app_user` VALUES (1,'alice@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','演示用户 Alice','虚构已启用账号，可用于本地 APP 登录',NULL,'active','','2026-01-01 00:00:00',NULL,NULL,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(2,'bob@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','演示用户 Bob','虚构已启用账号，用于会员和反馈展示',NULL,'active','','2026-01-01 00:00:00',NULL,NULL,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(3,'pending@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','待审核用户','虚构待审核账号',NULL,'pending','','2026-01-01 00:00:00',NULL,NULL,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(4,'disabled@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','停用用户','虚构停用账号',NULL,'disabled','演示：后台停用','2026-01-01 00:00:00',NULL,NULL,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(5,'rejected@example.test','email','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','驳回用户','虚构审核驳回账号',NULL,'rejected','演示：注册资料需要补充','2026-01-01 00:00:00',NULL,NULL,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL);
/*!40000 ALTER TABLE `app_user` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `casbin_rule`
--

DROP TABLE IF EXISTS `casbin_rule`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `casbin_rule` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `ptype` varchar(8) NOT NULL COMMENT 'p=策略 g=角色分组',
  `v0` varchar(128) NOT NULL DEFAULT '' COMMENT 'p: sub / g: user',
  `v1` varchar(128) NOT NULL DEFAULT '' COMMENT 'p: dom / g: role',
  `v2` varchar(255) NOT NULL DEFAULT '' COMMENT 'p: obj / g: dom',
  `v3` varchar(64) NOT NULL DEFAULT '' COMMENT 'p: act',
  `v4` varchar(64) NOT NULL DEFAULT '',
  `v5` varchar(64) NOT NULL DEFAULT '',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_ptype_v0_v1_v2_v3` (`ptype`,`v0`,`v1`,`v2`,`v3`),
  KEY `idx_v0` (`v0`),
  KEY `idx_v1` (`v1`)
) ENGINE=InnoDB AUTO_INCREMENT=51 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='Casbin 策略';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `casbin_rule`
--

LOCK TABLES `casbin_rule` WRITE;
/*!40000 ALTER TABLE `casbin_rule` DISABLE KEYS */;
INSERT INTO `casbin_rule` VALUES (1,'p','super_admin','*','/*','*','','','2026-01-01 00:00:00'),(2,'g','user:1','super_admin','*','','','','2026-01-01 00:00:00'),(3,'p','scope_all','*','/v1/sys/user/list','GET','','','2026-01-01 00:00:00'),(4,'p','scope_all','*','/v1/sys/user/:id','GET','','','2026-01-01 00:00:00'),(5,'p','scope_all','*','/v1/sys/dept/tree','GET','','','2026-01-01 00:00:00'),(6,'p','scope_all','*','/v1/sys/menu/my','GET','','','2026-01-01 00:00:00'),(7,'p','scope_all','*','/v1/sys/menu/my/perm-codes','GET','','','2026-01-01 00:00:00'),(8,'p','scope_all','*','/v1/sys/profile','GET','','','2026-01-01 00:00:00'),(9,'p','scope_all','*','/v1/sys/profile/','GET','','','2026-01-01 00:00:00'),(10,'p','scope_all','*','/v1/sys/profile','PUT','','','2026-01-01 00:00:00'),(11,'p','scope_all','*','/v1/sys/profile/','PUT','','','2026-01-01 00:00:00'),(12,'p','scope_all','*','/v1/sys/user/password/change','POST','','','2026-01-01 00:00:00'),(13,'p','scope_dept_children','*','/v1/sys/user/list','GET','','','2026-01-01 00:00:00'),(14,'p','scope_dept_children','*','/v1/sys/user/:id','GET','','','2026-01-01 00:00:00'),(15,'p','scope_dept_children','*','/v1/sys/dept/tree','GET','','','2026-01-01 00:00:00'),(16,'p','scope_dept_children','*','/v1/sys/menu/my','GET','','','2026-01-01 00:00:00'),(17,'p','scope_dept_children','*','/v1/sys/menu/my/perm-codes','GET','','','2026-01-01 00:00:00'),(18,'p','scope_dept_children','*','/v1/sys/profile','GET','','','2026-01-01 00:00:00'),(19,'p','scope_dept_children','*','/v1/sys/profile/','GET','','','2026-01-01 00:00:00'),(20,'p','scope_dept_children','*','/v1/sys/profile','PUT','','','2026-01-01 00:00:00'),(21,'p','scope_dept_children','*','/v1/sys/profile/','PUT','','','2026-01-01 00:00:00'),(22,'p','scope_dept_children','*','/v1/sys/user/password/change','POST','','','2026-01-01 00:00:00'),(23,'p','scope_dept','*','/v1/sys/user/list','GET','','','2026-01-01 00:00:00'),(24,'p','scope_dept','*','/v1/sys/user/:id','GET','','','2026-01-01 00:00:00'),(25,'p','scope_dept','*','/v1/sys/dept/tree','GET','','','2026-01-01 00:00:00'),(26,'p','scope_dept','*','/v1/sys/menu/my','GET','','','2026-01-01 00:00:00'),(27,'p','scope_dept','*','/v1/sys/menu/my/perm-codes','GET','','','2026-01-01 00:00:00'),(28,'p','scope_dept','*','/v1/sys/profile','GET','','','2026-01-01 00:00:00'),(29,'p','scope_dept','*','/v1/sys/profile/','GET','','','2026-01-01 00:00:00'),(30,'p','scope_dept','*','/v1/sys/profile','PUT','','','2026-01-01 00:00:00'),(31,'p','scope_dept','*','/v1/sys/profile/','PUT','','','2026-01-01 00:00:00'),(32,'p','scope_dept','*','/v1/sys/user/password/change','POST','','','2026-01-01 00:00:00'),(33,'p','scope_self','*','/v1/sys/user/list','GET','','','2026-01-01 00:00:00'),(34,'p','scope_self','*','/v1/sys/user/:id','GET','','','2026-01-01 00:00:00'),(35,'p','scope_self','*','/v1/sys/dept/tree','GET','','','2026-01-01 00:00:00'),(36,'p','scope_self','*','/v1/sys/menu/my','GET','','','2026-01-01 00:00:00'),(37,'p','scope_self','*','/v1/sys/menu/my/perm-codes','GET','','','2026-01-01 00:00:00'),(38,'p','scope_self','*','/v1/sys/profile','GET','','','2026-01-01 00:00:00'),(39,'p','scope_self','*','/v1/sys/profile/','GET','','','2026-01-01 00:00:00'),(40,'p','scope_self','*','/v1/sys/profile','PUT','','','2026-01-01 00:00:00'),(41,'p','scope_self','*','/v1/sys/profile/','PUT','','','2026-01-01 00:00:00'),(42,'p','scope_self','*','/v1/sys/user/password/change','POST','','','2026-01-01 00:00:00'),(43,'g','user:2','scope_all','dept:2','','','','2026-01-01 00:00:00'),(44,'g','user:3','scope_dept_children','dept:2','','','','2026-01-01 00:00:00'),(45,'g','user:3','scope_self','dept:5','','','','2026-01-01 00:00:00'),(46,'g','user:4','scope_dept','dept:3','','','','2026-01-01 00:00:00'),(47,'g','user:5','scope_self','dept:3','','','','2026-01-01 00:00:00'),(48,'g','user:6','scope_self','dept:4','','','','2026-01-01 00:00:00'),(49,'g','user:7','scope_self','dept:5','','','','2026-01-01 00:00:00'),(50,'g','user:8','scope_self','dept:5','','','','2026-01-01 00:00:00');
/*!40000 ALTER TABLE `casbin_rule` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_api`
--

DROP TABLE IF EXISTS `sys_api`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_api` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `group_name` varchar(64) NOT NULL COMMENT '分组（如 sys/user）',
  `method` varchar(8) NOT NULL COMMENT 'HTTP 方法：GET/POST/PUT/DELETE',
  `path` varchar(255) NOT NULL COMMENT '路径，如 /v1/sys/user/:id',
  `title` varchar(128) NOT NULL DEFAULT '' COMMENT '接口说明',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_method_path` (`method`,`path`)
) ENGINE=InnoDB AUTO_INCREMENT=111 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='API 资源';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_api`
--

LOCK TABLES `sys_api` WRITE;
/*!40000 ALTER TABLE `sys_api` DISABLE KEYS */;
INSERT INTO `sys_api` VALUES (1,'app/auth','POST','/v1/app/auth/email-code','POST /v1/app/auth/email-code','2026-01-01 00:00:00','2026-01-01 00:00:00'),(2,'app/auth','POST','/v1/app/auth/login','POST /v1/app/auth/login','2026-01-01 00:00:00','2026-01-01 00:00:00'),(3,'app/auth','POST','/v1/app/auth/logout','POST /v1/app/auth/logout','2026-01-01 00:00:00','2026-01-01 00:00:00'),(4,'app/auth','GET','/v1/app/auth/options','GET /v1/app/auth/options','2026-01-01 00:00:00','2026-01-01 00:00:00'),(5,'app/auth','POST','/v1/app/auth/refresh','POST /v1/app/auth/refresh','2026-01-01 00:00:00','2026-01-01 00:00:00'),(6,'app/auth','POST','/v1/app/auth/register','POST /v1/app/auth/register','2026-01-01 00:00:00','2026-01-01 00:00:00'),(7,'app/feedback','GET','/v1/app/feedback','GET /v1/app/feedback','2026-01-01 00:00:00','2026-01-01 00:00:00'),(8,'app/feedback','POST','/v1/app/feedback','POST /v1/app/feedback','2026-01-01 00:00:00','2026-01-01 00:00:00'),(9,'app/membership','GET','/v1/app/membership','GET /v1/app/membership','2026-01-01 00:00:00','2026-01-01 00:00:00'),(10,'app/profile','GET','/v1/app/profile','GET /v1/app/profile','2026-01-01 00:00:00','2026-01-01 00:00:00'),(11,'app/profile','PUT','/v1/app/profile','PUT /v1/app/profile','2026-01-01 00:00:00','2026-01-01 00:00:00'),(12,'sys/ai','GET','/v1/sys/ai/conversations','GET /v1/sys/ai/conversations','2026-01-01 00:00:00','2026-01-01 00:00:00'),(13,'sys/ai','POST','/v1/sys/ai/conversations','POST /v1/sys/ai/conversations','2026-01-01 00:00:00','2026-01-01 00:00:00'),(14,'sys/ai','DELETE','/v1/sys/ai/conversations/:id','DELETE /v1/sys/ai/conversations/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(15,'sys/ai','GET','/v1/sys/ai/conversations/:id','GET /v1/sys/ai/conversations/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(17,'sys/ai','GET','/v1/sys/ai/tasks','GET /v1/sys/ai/tasks','2026-01-01 00:00:00','2026-01-01 00:00:00'),(18,'sys/ai','GET','/v1/sys/ai/tasks/:id','GET /v1/sys/ai/tasks/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(19,'sys/ai','POST','/v1/sys/ai/tasks/:id/approve','POST /v1/sys/ai/tasks/:id/approve','2026-01-01 00:00:00','2026-01-01 00:00:00'),(20,'sys/ai','POST','/v1/sys/ai/tasks/:id/reject','POST /v1/sys/ai/tasks/:id/reject','2026-01-01 00:00:00','2026-01-01 00:00:00'),(21,'sys/ai','GET','/v1/sys/ai/tools','GET /v1/sys/ai/tools','2026-01-01 00:00:00','2026-01-01 00:00:00'),(22,'sys/ai','GET','/v1/sys/ai/workflows','GET /v1/sys/ai/workflows','2026-01-01 00:00:00','2026-01-01 00:00:00'),(23,'sys/ai','POST','/v1/sys/ai/workflows','POST /v1/sys/ai/workflows','2026-01-01 00:00:00','2026-01-01 00:00:00'),(24,'sys/ai','DELETE','/v1/sys/ai/workflows/:id','DELETE /v1/sys/ai/workflows/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(25,'sys/ai','GET','/v1/sys/ai/workflows/:id','GET /v1/sys/ai/workflows/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(26,'sys/ai','PUT','/v1/sys/ai/workflows/:id','PUT /v1/sys/ai/workflows/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(27,'sys/ai','POST','/v1/sys/ai/workflows/:id/run','POST /v1/sys/ai/workflows/:id/run','2026-01-01 00:00:00','2026-01-01 00:00:00'),(28,'sys/api','DELETE','/v1/sys/api/:id','DELETE /v1/sys/api/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(29,'sys/api','GET','/v1/sys/api/list','GET /v1/sys/api/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),(30,'sys/api','POST','/v1/sys/api/sync','POST /v1/sys/api/sync','2026-01-01 00:00:00','2026-01-01 00:00:00'),(31,'sys/app','GET','/v1/sys/app/feedback','GET /v1/sys/app/feedback','2026-01-01 00:00:00','2026-01-01 00:00:00'),(32,'sys/app','PUT','/v1/sys/app/feedback/:id','PUT /v1/sys/app/feedback/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(33,'sys/app','GET','/v1/sys/app/membership-plans','GET /v1/sys/app/membership-plans','2026-01-01 00:00:00','2026-01-01 00:00:00'),(34,'sys/app','GET','/v1/sys/app/memberships','GET /v1/sys/app/memberships','2026-01-01 00:00:00','2026-01-01 00:00:00'),(35,'sys/app','POST','/v1/sys/app/memberships','POST /v1/sys/app/memberships','2026-01-01 00:00:00','2026-01-01 00:00:00'),(36,'sys/app','DELETE','/v1/sys/app/memberships/:id','DELETE /v1/sys/app/memberships/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(37,'sys/app','GET','/v1/sys/app/users','GET /v1/sys/app/users','2026-01-01 00:00:00','2026-01-01 00:00:00'),(38,'sys/app','GET','/v1/sys/app/users/:id','GET /v1/sys/app/users/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(39,'sys/app','PUT','/v1/sys/app/users/:id','PUT /v1/sys/app/users/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(40,'sys/app','PUT','/v1/sys/app/users/:id/avatar','PUT /v1/sys/app/users/:id/avatar','2026-01-01 00:00:00','2026-01-01 00:00:00'),(41,'sys/app','POST','/v1/sys/app/users/:id/review','POST /v1/sys/app/users/:id/review','2026-01-01 00:00:00','2026-01-01 00:00:00'),(42,'sys/app','GET','/v1/sys/app/users/:id/sessions','GET /v1/sys/app/users/:id/sessions','2026-01-01 00:00:00','2026-01-01 00:00:00'),(43,'sys/app','DELETE','/v1/sys/app/users/:id/sessions/:session_id','DELETE /v1/sys/app/users/:id/sessions/:session_id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(44,'sys/audit','GET','/v1/sys/audit/list','GET /v1/sys/audit/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),(45,'sys/auth','GET','/v1/sys/auth/captcha','GET /v1/sys/auth/captcha','2026-01-01 00:00:00','2026-01-01 00:00:00'),(46,'sys/auth','POST','/v1/sys/auth/login','POST /v1/sys/auth/login','2026-01-01 00:00:00','2026-01-01 00:00:00'),(47,'sys/auth','POST','/v1/sys/auth/logout','POST /v1/sys/auth/logout','2026-01-01 00:00:00','2026-01-01 00:00:00'),(48,'sys/auth','POST','/v1/sys/auth/refresh','POST /v1/sys/auth/refresh','2026-01-01 00:00:00','2026-01-01 00:00:00'),(49,'sys/auth','GET','/v1/sys/auth/sessions','GET /v1/sys/auth/sessions','2026-01-01 00:00:00','2026-01-01 00:00:00'),(50,'sys/auth','DELETE','/v1/sys/auth/sessions/:id','DELETE /v1/sys/auth/sessions/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(51,'sys/dept','DELETE','/v1/sys/dept/:id','DELETE /v1/sys/dept/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(52,'sys/dept','PUT','/v1/sys/dept/:id','PUT /v1/sys/dept/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(53,'sys/dept','POST','/v1/sys/dept/create','POST /v1/sys/dept/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),(54,'sys/dept','GET','/v1/sys/dept/my','GET /v1/sys/dept/my','2026-01-01 00:00:00','2026-01-01 00:00:00'),(55,'sys/dept','GET','/v1/sys/dept/tree','GET /v1/sys/dept/tree','2026-01-01 00:00:00','2026-01-01 00:00:00'),(56,'sys/dict','DELETE','/v1/sys/dict/:id','DELETE /v1/sys/dict/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(57,'sys/dict','PUT','/v1/sys/dict/:id','PUT /v1/sys/dict/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(58,'sys/dict','GET','/v1/sys/dict/code/:code','GET /v1/sys/dict/code/:code','2026-01-01 00:00:00','2026-01-01 00:00:00'),(59,'sys/dict','POST','/v1/sys/dict/create','POST /v1/sys/dict/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),(60,'sys/dict','GET','/v1/sys/dict/list','GET /v1/sys/dict/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),(61,'sys/file','DELETE','/v1/sys/file/:id','DELETE /v1/sys/file/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(62,'sys/file','GET','/v1/sys/file/:id/url','GET /v1/sys/file/:id/url','2026-01-01 00:00:00','2026-01-01 00:00:00'),(63,'sys/file','GET','/v1/sys/file/list','GET /v1/sys/file/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),(64,'sys/file','POST','/v1/sys/file/upload','POST /v1/sys/file/upload','2026-01-01 00:00:00','2026-01-01 00:00:00'),(65,'sys/menu','DELETE','/v1/sys/menu/:id','DELETE /v1/sys/menu/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(66,'sys/menu','GET','/v1/sys/menu/:id','GET /v1/sys/menu/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(67,'sys/menu','PUT','/v1/sys/menu/:id','PUT /v1/sys/menu/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(68,'sys/menu','POST','/v1/sys/menu/create','POST /v1/sys/menu/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),(69,'sys/menu','GET','/v1/sys/menu/list','GET /v1/sys/menu/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),(70,'sys/menu','GET','/v1/sys/menu/my','GET /v1/sys/menu/my','2026-01-01 00:00:00','2026-01-01 00:00:00'),(71,'sys/menu','GET','/v1/sys/menu/my/perm-codes','GET /v1/sys/menu/my/perm-codes','2026-01-01 00:00:00','2026-01-01 00:00:00'),(72,'sys/menu','GET','/v1/sys/menu/tree','GET /v1/sys/menu/tree','2026-01-01 00:00:00','2026-01-01 00:00:00'),(73,'sys/profile','GET','/v1/sys/profile','GET /v1/sys/profile','2026-01-01 00:00:00','2026-01-01 00:00:00'),(74,'sys/profile','PUT','/v1/sys/profile','PUT /v1/sys/profile','2026-01-01 00:00:00','2026-01-01 00:00:00'),(75,'sys/role','DELETE','/v1/sys/role/:id','DELETE /v1/sys/role/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(76,'sys/role','GET','/v1/sys/role/:id','GET /v1/sys/role/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(77,'sys/role','PUT','/v1/sys/role/:id','PUT /v1/sys/role/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(78,'sys/role','POST','/v1/sys/role/:id/menus','POST /v1/sys/role/:id/menus','2026-01-01 00:00:00','2026-01-01 00:00:00'),(79,'sys/role','POST','/v1/sys/role/create','POST /v1/sys/role/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),(80,'sys/role','GET','/v1/sys/role/list','GET /v1/sys/role/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),(81,'sys/tenant','DELETE','/v1/sys/tenant/:id','DELETE /v1/sys/tenant/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(82,'sys/tenant','GET','/v1/sys/tenant/:id','GET /v1/sys/tenant/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(83,'sys/tenant','POST','/v1/sys/tenant/:id/reset-admin','POST /v1/sys/tenant/:id/reset-admin','2026-01-01 00:00:00','2026-01-01 00:00:00'),(84,'sys/tenant','POST','/v1/sys/tenant/:id/start','POST /v1/sys/tenant/:id/start','2026-01-01 00:00:00','2026-01-01 00:00:00'),(85,'sys/tenant','POST','/v1/sys/tenant/:id/stop','POST /v1/sys/tenant/:id/stop','2026-01-01 00:00:00','2026-01-01 00:00:00'),(86,'sys/tenant','POST','/v1/sys/tenant/create','POST /v1/sys/tenant/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),(87,'sys/tenant','GET','/v1/sys/tenant/list','GET /v1/sys/tenant/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),(88,'sys/user','DELETE','/v1/sys/user/:id','DELETE /v1/sys/user/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(89,'sys/user','GET','/v1/sys/user/:id','GET /v1/sys/user/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(90,'sys/user','PUT','/v1/sys/user/:id','PUT /v1/sys/user/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(91,'sys/user','POST','/v1/sys/user/:id/password','POST /v1/sys/user/:id/password','2026-01-01 00:00:00','2026-01-01 00:00:00'),(92,'sys/user','POST','/v1/sys/user/create','POST /v1/sys/user/create','2026-01-01 00:00:00','2026-01-01 00:00:00'),(93,'sys/user','GET','/v1/sys/user/list','GET /v1/sys/user/list','2026-01-01 00:00:00','2026-01-01 00:00:00'),(94,'sys/user','POST','/v1/sys/user/password/change','POST /v1/sys/user/password/change','2026-01-01 00:00:00','2026-01-01 00:00:00'),(95,'sys/ai','POST','/v1/sys/ai/messages/stream','POST /v1/sys/ai/messages/stream','2026-01-01 00:00:00','2026-01-01 00:00:00'),(96,'sys/ai','GET','/v1/sys/ai/kbs','GET /v1/sys/ai/kbs','2026-01-01 00:00:00','2026-01-01 00:00:00'),(97,'sys/ai','POST','/v1/sys/ai/kbs','POST /v1/sys/ai/kbs','2026-01-01 00:00:00','2026-01-01 00:00:00'),(98,'sys/ai','GET','/v1/sys/ai/kbs/:id','GET /v1/sys/ai/kbs/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(99,'sys/ai','PUT','/v1/sys/ai/kbs/:id','PUT /v1/sys/ai/kbs/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(100,'sys/ai','DELETE','/v1/sys/ai/kbs/:id','DELETE /v1/sys/ai/kbs/:id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(101,'sys/ai','GET','/v1/sys/ai/kbs/:id/docs','GET /v1/sys/ai/kbs/:id/docs','2026-01-01 00:00:00','2026-01-01 00:00:00'),(102,'sys/ai','POST','/v1/sys/ai/kbs/:id/docs','POST /v1/sys/ai/kbs/:id/docs','2026-01-01 00:00:00','2026-01-01 00:00:00'),(103,'sys/ai','DELETE','/v1/sys/ai/kbs/:id/docs/:doc_id','DELETE /v1/sys/ai/kbs/:id/docs/:doc_id','2026-01-01 00:00:00','2026-01-01 00:00:00'),(104,'sys/ai','POST','/v1/sys/ai/kbs/:id/docs/:doc_id/retry','POST /v1/sys/ai/kbs/:id/docs/:doc_id/retry','2026-01-01 00:00:00','2026-01-01 00:00:00'),(105,'sys/ai','POST','/v1/sys/ai/kbs/:id/search','POST /v1/sys/ai/kbs/:id/search','2026-11-07 00:23:53','2026-11-07 00:23:53'),(106,'sys/ai','GET','/v1/sys/ai/kbs/:id/docs/:doc_id','GET /v1/sys/ai/kbs/:id/docs/:doc_id','2026-11-07 01:34:42','2026-11-07 01:34:42'),(107,'sys/ai','POST','/v1/sys/ai/tasks/:id/cancel','POST /v1/sys/ai/tasks/:id/cancel','2026-11-07 01:34:42','2026-11-07 01:34:42'),(108,'sys/ai','PUT','/v1/sys/ai/conversations/:id','PUT /v1/sys/ai/conversations/:id','2026-11-07 01:34:42','2026-11-07 01:34:42'),(109,'sys/ai','PUT','/v1/sys/ai/conversations/:id/messages/:message_id/feedback','PUT /v1/sys/ai/conversations/:id/messages/:message_id/feedback','2026-11-07 01:34:42','2026-11-07 01:34:42'),(110,'sys/ai','PUT','/v1/sys/ai/kbs/:id/docs/:doc_id','PUT /v1/sys/ai/kbs/:id/docs/:doc_id','2026-11-07 01:34:42','2026-11-07 01:34:42');
/*!40000 ALTER TABLE `sys_api` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_audit_log`
--

DROP TABLE IF EXISTS `sys_audit_log`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_audit_log` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `user_id` bigint unsigned DEFAULT NULL COMMENT '操作人ID',
  `username` varchar(64) NOT NULL DEFAULT '' COMMENT '操作人用户名',
  `module` varchar(64) NOT NULL COMMENT '模块（如 sys/user）',
  `action` varchar(64) NOT NULL COMMENT '操作（如 create/update/delete/login）',
  `method` varchar(8) NOT NULL DEFAULT '' COMMENT 'HTTP 方法',
  `path` varchar(255) NOT NULL DEFAULT '' COMMENT '请求路径',
  `params` text COMMENT '请求参数（JSON，敏感字段脱敏）',
  `result` tinyint NOT NULL DEFAULT '1' COMMENT '结果：1成功 2失败',
  `error_msg` varchar(512) NOT NULL DEFAULT '' COMMENT '错误信息',
  `ip` varchar(64) NOT NULL DEFAULT '' COMMENT '客户端IP',
  `user_agent` varchar(512) NOT NULL DEFAULT '' COMMENT 'User-Agent',
  `duration_ms` int unsigned NOT NULL DEFAULT '0' COMMENT '耗时毫秒',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  KEY `idx_user` (`user_id`),
  KEY `idx_module_action` (`module`,`action`),
  KEY `idx_created` (`created_at`)
) ENGINE=InnoDB AUTO_INCREMENT=102 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='审计日志';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_audit_log`
--

LOCK TABLES `sys_audit_log` WRITE;
/*!40000 ALTER TABLE `sys_audit_log` DISABLE KEYS */;
INSERT INTO `sys_audit_log` VALUES (1,NULL,'','sys/auth','refresh','POST','/v1/sys/auth/refresh',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36',29,'2026-11-06 22:50:23'),(2,1,'admin','sys/auth','login','POST','/v1/sys/auth/login',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36',184,'2026-11-06 22:50:27'),(3,NULL,'','sys/auth','login','POST','/v1/sys/auth/login',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',211,'2026-11-06 23:38:00'),(4,NULL,'','sys/auth','login','POST','/v1/sys/auth/login',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',86,'2026-11-06 23:41:26'),(5,NULL,'','sys/auth','login','POST','/v1/sys/auth/login',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',3,'2026-11-06 23:43:59'),(6,NULL,'','sys/auth','login','POST','/v1/sys/auth/login',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',88,'2026-11-06 23:44:43'),(7,1,'admin','sys/auth','login','POST','/v1/sys/auth/login',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',423,'2026-11-06 23:52:14'),(8,1,'admin','sys/auth','login','POST','/v1/sys/auth/login',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',143,'2026-11-06 23:54:03'),(9,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs',NULL,1,'','172.20.0.1','node',47,'2026-11-06 23:55:52'),(10,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/docs',NULL,1,'','172.20.0.1','node',1068,'2026-11-06 23:55:53'),(11,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/docs',NULL,1,'','172.20.0.1','node',784,'2026-11-06 23:55:54'),(12,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/docs',NULL,1,'','172.20.0.1','node',419,'2026-11-06 23:55:55'),(13,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/docs',NULL,1,'','172.20.0.1','node',642,'2026-11-06 23:55:55'),(14,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/docs',NULL,1,'','172.20.0.1','node',356,'2026-11-06 23:55:56'),(15,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/docs',NULL,1,'','172.20.0.1','node',414,'2026-11-06 23:55:56'),(16,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs',NULL,1,'','172.20.0.1','node',67,'2026-11-06 23:55:56'),(17,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/docs',NULL,1,'','172.20.0.1','node',280,'2026-11-06 23:55:57'),(18,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/docs',NULL,1,'','172.20.0.1','node',295,'2026-11-06 23:55:57'),(19,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/docs',NULL,1,'','172.20.0.1','node',294,'2026-11-06 23:55:57'),(20,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/docs',NULL,1,'','172.20.0.1','node',308,'2026-11-06 23:55:58'),(21,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/docs',NULL,1,'','172.20.0.1','node',310,'2026-11-06 23:55:58'),(22,1,'admin','sys/ai','post','POST','/v1/sys/ai/messages/stream',NULL,1,'','172.20.0.1','node',4447,'2026-11-06 23:56:14'),(23,1,'admin','sys/ai','post','POST','/v1/sys/ai/messages/stream',NULL,1,'','172.20.0.1','node',4587,'2026-11-06 23:56:18'),(24,1,'admin','sys/ai','post','POST','/v1/sys/ai/messages/stream',NULL,1,'','172.20.0.1','node',3418,'2026-11-06 23:56:22'),(25,1,'admin','sys/ai','post','POST','/v1/sys/ai/messages/stream',NULL,1,'','172.20.0.1','node',4310,'2026-11-06 23:56:26'),(26,1,'admin','sys/ai','post','POST','/v1/sys/ai/messages/stream',NULL,1,'','172.20.0.1','node',2564,'2026-11-06 23:56:29'),(27,1,'admin','sys/ai','post','POST','/v1/sys/ai/messages/stream',NULL,1,'','172.20.0.1','node',3582,'2026-11-06 23:56:32'),(28,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/docs',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',405,'2026-11-06 23:59:13'),(29,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/docs',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',12,'2026-11-06 23:59:15'),(30,1,'admin','sys/ai','delete','DELETE','/v1/sys/ai/kbs/2/docs/12',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',76,'2026-11-07 00:00:18'),(31,1,'admin','sys/auth','logout','POST','/v1/sys/auth/logout',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',30,'2026-11-07 00:02:47'),(32,1,'admin','sys/auth','logout','POST','/v1/sys/auth/logout',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',9,'2026-11-07 00:02:49'),(33,1,'admin','sys/auth','login','POST','/v1/sys/auth/login',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',266,'2026-11-07 00:12:05'),(34,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',71,'2026-11-07 00:22:10'),(35,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',5,'2026-11-07 00:22:11'),(36,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/999999999/search',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',27,'2026-11-07 00:22:11'),(37,NULL,'','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',0,'2026-11-07 00:22:11'),(38,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,1,'','172.20.0.1','node',370,'2026-11-07 00:22:29'),(39,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,1,'','172.20.0.1','node',211,'2026-11-07 00:22:30'),(40,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,1,'','172.20.0.1','node',178,'2026-11-07 00:22:30'),(41,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,1,'','172.20.0.1','node',180,'2026-11-07 00:22:30'),(42,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,1,'','172.20.0.1','node',225,'2026-11-07 00:22:30'),(43,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,1,'','172.20.0.1','node',182,'2026-11-07 00:22:31'),(44,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/search',NULL,1,'','172.20.0.1','node',278,'2026-11-07 00:22:31'),(45,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/search',NULL,1,'','172.20.0.1','node',304,'2026-11-07 00:22:31'),(46,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/search',NULL,1,'','172.20.0.1','node',279,'2026-11-07 00:22:32'),(47,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/search',NULL,1,'','172.20.0.1','node',178,'2026-11-07 00:22:32'),(48,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/search',NULL,1,'','172.20.0.1','node',153,'2026-11-07 00:22:32'),(49,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/search',NULL,1,'','172.20.0.1','node',175,'2026-11-07 00:22:32'),(50,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,1,'','172.20.0.1','node',199,'2026-11-07 00:22:33'),(51,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/search',NULL,1,'','172.20.0.1','node',204,'2026-11-07 00:22:33'),(52,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36',185,'2026-11-07 00:23:14'),(53,1,'admin','sys/api','post','POST','/v1/sys/api/sync',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',155,'2026-11-07 00:23:53'),(54,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36',158,'2026-11-07 00:23:54'),(55,1,'admin','sys/auth','login','POST','/v1/sys/auth/login',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',124,'2026-11-07 00:24:39'),(56,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/2/search',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',187,'2026-11-07 00:24:42'),(57,2,'demo_all','sys/auth','login','POST','/v1/sys/auth/login',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',110,'2026-11-07 00:25:26'),(58,2,'demo_all','sys/ai','post','POST','/v1/sys/ai/kbs/1/search',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',12,'2026-11-07 00:25:26'),(59,2,'demo_all','sys/auth','logout','POST','/v1/sys/auth/logout',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',13,'2026-11-07 00:25:26'),(60,1,'admin','sys/auth','logout','POST','/v1/sys/auth/logout',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',8,'2026-11-07 00:25:40'),(61,1,'admin','sys/auth','logout','POST','/v1/sys/auth/logout',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',10,'2026-11-07 00:28:11'),(62,1,'admin','sys/auth','login','POST','/v1/sys/auth/login',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',230,'2026-11-07 00:52:44'),(63,1,'admin','sys/auth','login','POST','/v1/sys/auth/login',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',129,'2026-11-07 00:56:33'),(64,1,'admin','sys/ai','post','POST','/v1/sys/ai/workflows',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',83,'2026-11-07 00:56:38'),(65,1,'admin','sys/ai','post','POST','/v1/sys/ai/workflows/1/run',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',97,'2026-11-07 00:59:03'),(66,1,'admin','sys/ai','post','POST','/v1/sys/ai/tasks/1/approve',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',3404,'2026-11-07 01:00:54'),(67,1,'admin','sys/ai','post','POST','/v1/sys/ai/tasks/1/approve',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',355,'2026-11-07 01:01:31'),(68,1,'admin','sys/ai','put','PUT','/v1/sys/ai/workflows/1',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',51,'2026-11-07 01:03:18'),(69,1,'admin','sys/ai','delete','DELETE','/v1/sys/ai/workflows/1',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',52,'2026-11-07 01:05:58'),(70,1,'admin','sys/auth','logout','POST','/v1/sys/auth/logout',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',353,'2026-11-07 01:08:14'),(71,1,'admin','sys/auth','logout','POST','/v1/sys/auth/logout',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Microsoft Windows 10.0.26300; zh-CN) PowerShell/7.6.5',7,'2026-11-07 01:09:04'),(72,1,'admin','sys/auth','refresh','POST','/v1/sys/auth/refresh',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36',28,'2026-11-07 01:19:01'),(73,1,'admin','sys/auth','login','POST','/v1/sys/auth/login',NULL,1,'','172.20.0.1','node',151,'2026-11-07 01:25:30'),(74,1,'admin','sys/menu','post','POST','/v1/sys/menu/create',NULL,1,'','172.20.0.1','node',58,'2026-11-07 01:25:31'),(75,1,'admin','sys/role','post','POST','/v1/sys/role/1/menus',NULL,1,'','172.20.0.1','node',145,'2026-11-07 01:25:31'),(76,1,'admin','sys/api','post','POST','/v1/sys/api/sync',NULL,1,'','172.20.0.1','node',266,'2026-11-07 01:34:42'),(77,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',425,'2026-11-07 01:36:52'),(78,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/3/docs',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',1697,'2026-11-07 01:37:38'),(79,1,'admin','sys/ai','post','POST','/v1/sys/ai/kbs/3/docs',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',1750,'2026-11-07 01:37:38'),(80,1,'admin','sys/ai','put','PUT','/v1/sys/ai/kbs/3/docs/13',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',865,'2026-11-07 01:38:17'),(81,1,'admin','sys/ai','post','POST','/v1/sys/ai/workflows',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',389,'2026-11-07 01:39:51'),(82,1,'admin','sys/ai','post','POST','/v1/sys/ai/workflows',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/156.0.0.0 Safari/537.36',412,'2026-11-07 01:41:04'),(83,1,'admin','sys/ai','post','POST','/v1/sys/ai/messages/stream',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',3684,'2026-11-07 01:41:54'),(84,1,'admin','sys/ai','put','PUT','/v1/sys/ai/conversations/7/messages/14/feedback',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',12,'2026-11-07 01:42:32'),(85,1,'admin','sys/ai','post','POST','/v1/sys/ai/workflows/2/run',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',393,'2026-11-07 01:44:30'),(86,1,'admin','sys/ai','post','POST','/v1/sys/ai/tasks/2/approve',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',50,'2026-11-07 01:44:32'),(87,NULL,'','sys/auth','refresh','POST','/v1/sys/auth/refresh',NULL,2,'请求失败','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',130,'2026-11-07 01:45:24'),(88,1,'admin','sys/ai','post','POST','/v1/sys/ai/workflows/2/run',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',239,'2026-11-07 01:49:14'),(89,1,'admin','sys/ai','post','POST','/v1/sys/ai/tasks/3/cancel',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',61,'2026-11-07 01:49:16'),(90,1,'admin','sys/ai','put','PUT','/v1/sys/ai/conversations/7/messages/14/feedback',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',405,'2026-11-07 01:53:24'),(91,1,'admin','sys/ai','put','PUT','/v1/sys/ai/conversations/7',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',391,'2026-11-07 01:53:27'),(92,1,'admin','sys/ai','put','PUT','/v1/sys/ai/conversations/7',NULL,1,'','172.20.0.1','Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/156.0.0.0 Safari/537.36',225,'2026-11-07 01:53:28'),(93,1,'admin','sys/ai','put','PUT','/v1/sys/ai/kbs/3/docs/13',NULL,2,'请求失败','172.20.0.1','node',18,'2026-11-07 01:54:24'),(94,1,'admin','sys/ai','post','POST','/v1/sys/ai/workflows/2/run',NULL,1,'','172.20.0.1','node',13,'2026-11-07 01:54:24'),(95,1,'admin','sys/ai','post','POST','/v1/sys/ai/workflows/2/run',NULL,2,'请求失败','172.20.0.1','node',11,'2026-11-07 01:54:24'),(96,1,'admin','sys/ai','delete','DELETE','/v1/sys/ai/conversations/7',NULL,1,'','172.20.0.1','node',47,'2026-11-07 01:56:04'),(97,1,'admin','sys/ai','delete','DELETE','/v1/sys/ai/kbs/3',NULL,1,'','172.20.0.1','node',171,'2026-11-07 01:56:04'),(98,1,'admin','sys/ai','delete','DELETE','/v1/sys/ai/workflows/2',NULL,1,'','172.20.0.1','node',78,'2026-11-07 01:56:04'),(99,1,'admin','sys/auth','logout','POST','/v1/sys/auth/logout',NULL,1,'','172.20.0.1','node',387,'2026-11-07 01:56:39'),(100,1,'admin','sys/auth','login','POST','/v1/sys/auth/login',NULL,1,'','172.20.0.1','node',180,'2026-11-07 11:42:26'),(101,1,'admin','sys/auth','logout','POST','/v1/sys/auth/logout',NULL,1,'','172.20.0.1','node',26,'2026-11-07 11:42:26');
/*!40000 ALTER TABLE `sys_audit_log` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_dept`
--

DROP TABLE IF EXISTS `sys_dept`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_dept` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `parent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '父部门ID，0=根',
  `name` varchar(64) NOT NULL COMMENT '部门名称',
  `leader` varchar(64) NOT NULL DEFAULT '' COMMENT '负责人',
  `leader_user_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '负责人账号ID，0=未关联',
  `phone` varchar(20) NOT NULL DEFAULT '' COMMENT '联系电话',
  `email` varchar(128) NOT NULL DEFAULT '' COMMENT '邮箱',
  `sort` int NOT NULL DEFAULT '0' COMMENT '排序（小在前）',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1启用 2停用',
  `path` varchar(512) NOT NULL COMMENT '物化路径，如 /1/7/12/，根为 /1/',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  KEY `idx_parent` (`parent_id`),
  KEY `idx_path` (`path`(191))
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='部门';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_dept`
--

LOCK TABLES `sys_dept` WRITE;
/*!40000 ALTER TABLE `sys_dept` DISABLE KEYS */;
INSERT INTO `sys_dept` VALUES (1,0,'总部','',0,'','',0,1,'/1/','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(2,1,'研发中心','研发负责人',3,'','rd@example.test',10,1,'/1/2/','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(3,2,'平台研发组','平台负责人',4,'','platform@example.test',10,1,'/1/2/3/','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(4,2,'质量保障组','',0,'','qa@example.test',20,1,'/1/2/4/','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(5,1,'运营中心','运营负责人',7,'','operations@example.test',20,1,'/1/5/','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(6,1,'归档示例部门','',0,'','',99,2,'/1/6/','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL);
/*!40000 ALTER TABLE `sys_dept` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_dict`
--

DROP TABLE IF EXISTS `sys_dict`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_dict` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(64) NOT NULL COMMENT '字典名称',
  `code` varchar(64) NOT NULL COMMENT '字典编码（英文唯一）',
  `items` json NOT NULL COMMENT '[{"label":"启用","value":1},{"label":"停用","value":2}]',
  `description` varchar(512) NOT NULL DEFAULT '' COMMENT '描述',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1启用 2停用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`)
) ENGINE=InnoDB AUTO_INCREMENT=5 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='字典';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_dict`
--

LOCK TABLES `sys_dict` WRITE;
/*!40000 ALTER TABLE `sys_dict` DISABLE KEYS */;
INSERT INTO `sys_dict` VALUES (1,'启停状态','user_status','[{\"sort\": 0, \"label\": \"启用\", \"value\": 1}, {\"sort\": 1, \"label\": \"停用\", \"value\": 2}]','后台用户、部门和角色状态',1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(2,'用户性别','user_gender','[{\"sort\": 0, \"label\": \"未知\", \"value\": 0}, {\"sort\": 1, \"label\": \"男\", \"value\": 1}, {\"sort\": 2, \"label\": \"女\", \"value\": 2}]','后台用户性别',1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(3,'数据范围','data_scope','[{\"sort\": 0, \"label\": \"全部\", \"value\": \"ALL\"}, {\"sort\": 1, \"label\": \"本部门及下级\", \"value\": \"DEPT_AND_CHILDREN\"}, {\"sort\": 2, \"label\": \"本部门\", \"value\": \"DEPT\"}, {\"sort\": 3, \"label\": \"本人\", \"value\": \"SELF\"}]','用户数据查询范围，多角色取并集',1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(4,'APP会员套餐','app_membership_plan','[{\"sort\": 0, \"label\": \"基础会员\", \"value\": \"basic\"}, {\"sort\": 1, \"label\": \"专业会员\", \"value\": \"pro\"}]','APP 后台开通会员时的套餐选择',1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL);
/*!40000 ALTER TABLE `sys_dict` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_file`
--

DROP TABLE IF EXISTS `sys_file`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_file` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `bucket` varchar(64) NOT NULL COMMENT 'bucket 名',
  `object_key` varchar(512) NOT NULL COMMENT '对象键（存储路径）',
  `original_name` varchar(255) NOT NULL DEFAULT '' COMMENT '原始文件名',
  `ext` varchar(16) NOT NULL DEFAULT '' COMMENT '扩展名',
  `mime` varchar(128) NOT NULL DEFAULT '' COMMENT 'MIME 类型',
  `size` bigint unsigned NOT NULL DEFAULT '0' COMMENT '字节数',
  `visibility` varchar(16) NOT NULL DEFAULT 'public' COMMENT '可见性：public/private',
  `ref_count` int NOT NULL DEFAULT '0' COMMENT '引用计数（0 可删）',
  `uploader_id` bigint unsigned DEFAULT NULL COMMENT '上传者用户ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  KEY `idx_bucket_key` (`bucket`,`object_key`(191)),
  KEY `idx_uploader` (`uploader_id`),
  KEY `idx_ref_count` (`ref_count`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文件资源';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_file`
--

LOCK TABLES `sys_file` WRITE;
/*!40000 ALTER TABLE `sys_file` DISABLE KEYS */;
/*!40000 ALTER TABLE `sys_file` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_file_reference`
--

DROP TABLE IF EXISTS `sys_file_reference`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_file_reference` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `file_id` bigint unsigned NOT NULL COMMENT '文件ID',
  `ref_type` varchar(32) NOT NULL COMMENT '业务引用类型：admin_avatar/app_avatar',
  `ref_id` bigint unsigned NOT NULL COMMENT '业务对象ID',
  `created_by` bigint unsigned NOT NULL COMMENT '建立引用的操作者ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_file_ref` (`file_id`,`ref_type`,`ref_id`),
  KEY `idx_ref_subject` (`ref_type`,`ref_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='文件业务引用明细';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_file_reference`
--

LOCK TABLES `sys_file_reference` WRITE;
/*!40000 ALTER TABLE `sys_file_reference` DISABLE KEYS */;
/*!40000 ALTER TABLE `sys_file_reference` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_menu`
--

DROP TABLE IF EXISTS `sys_menu`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_menu` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `parent_id` bigint unsigned NOT NULL DEFAULT '0' COMMENT '父菜单ID，0=根',
  `title` varchar(64) NOT NULL COMMENT '显示名称',
  `name` varchar(64) NOT NULL DEFAULT '' COMMENT '路由名（英文，前端用）',
  `type` tinyint NOT NULL COMMENT '类型：1目录 2菜单 3按钮',
  `path` varchar(255) NOT NULL DEFAULT '' COMMENT '路由路径',
  `component` varchar(255) NOT NULL DEFAULT '' COMMENT '前端组件路径',
  `perm_code` varchar(128) NOT NULL DEFAULT '' COMMENT '按钮权限码（如 sys:user:add）',
  `icon` varchar(64) NOT NULL DEFAULT '' COMMENT '图标',
  `sort` int NOT NULL DEFAULT '0' COMMENT '排序',
  `visible` tinyint NOT NULL DEFAULT '1' COMMENT '是否显示：1显示 0隐藏',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1启用 2停用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  KEY `idx_parent` (`parent_id`),
  KEY `idx_type` (`type`)
) ENGINE=InnoDB AUTO_INCREMENT=9001446 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='菜单';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_menu`
--

LOCK TABLES `sys_menu` WRITE;
/*!40000 ALTER TABLE `sys_menu` DISABLE KEYS */;
INSERT INTO `sys_menu` VALUES (9001300,0,'系统管理','system',1,'/sys','','','system',10,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001301,0,'工作台','dashboard',2,'/dashboard','dashboard','','dashboard',0,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001302,9001300,'用户管理','user',2,'/sys/user','user','','user',20,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001303,9001300,'角色管理','role',2,'/sys/role','role','','role',21,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001304,9001300,'菜单管理','menu',2,'/sys/menu','menu','','menu',22,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001305,9001300,'部门管理','dept',2,'/sys/dept','dept','','dept',23,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001306,9001300,'字典管理','dict',2,'/sys/dict','dict','','dict',24,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001307,9001300,'API 资源','api',2,'/sys/api','api','','api',25,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001308,9001300,'文件管理','file',2,'/sys/file','file','','file',26,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001309,9001300,'审计日志','audit',2,'/sys/audit','audit','','audit',27,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001310,9001300,'个人中心','profile',2,'/sys/profile','profile','','profile',28,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001320,9001302,'新增用户','user_add',3,'','','sys:user:add','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001321,9001302,'编辑用户','user_edit',3,'','','sys:user:edit','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001322,9001302,'删除用户','user_delete',3,'','','sys:user:delete','',2,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001323,9001302,'重置密码','user_reset_password',3,'','','sys:user:reset-password','',3,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001324,9001303,'新增角色','role_add',3,'','','sys:role:add','',4,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001325,9001303,'编辑角色','role_edit',3,'','','sys:role:edit','',5,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001326,9001303,'删除角色','role_delete',3,'','','sys:role:delete','',6,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001327,9001303,'分配菜单','role_menus',3,'','','sys:role:menus','',7,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001328,9001304,'新增菜单','menu_add',3,'','','sys:menu:add','',8,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001329,9001304,'编辑菜单','menu_edit',3,'','','sys:menu:edit','',9,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001330,9001304,'删除菜单','menu_delete',3,'','','sys:menu:delete','',10,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001331,9001305,'新增部门','dept_add',3,'','','sys:dept:add','',11,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001332,9001305,'编辑部门','dept_edit',3,'','','sys:dept:edit','',12,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001333,9001305,'删除部门','dept_delete',3,'','','sys:dept:delete','',13,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001334,9001306,'新增字典','dict_add',3,'','','sys:dict:add','',14,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001335,9001306,'编辑字典','dict_edit',3,'','','sys:dict:edit','',15,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001336,9001306,'删除字典','dict_delete',3,'','','sys:dict:delete','',16,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001337,9001307,'删除 API','api_delete',3,'','','sys:api:delete','',17,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001338,9001308,'删除文件','file_delete',3,'','','sys:file:delete','',18,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001339,9001310,'编辑资料','profile_edit',3,'','','sys:profile:edit','',19,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001360,0,'APP管理','app',1,'/app','','','app',20,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001361,9001360,'APP用户','app_user',2,'/app/users','app_user','','app_user',0,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001362,9001360,'会员管理','app_membership',2,'/app/memberships','app_membership','','membership',1,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001363,9001361,'审核用户','app_user_review',3,'','','sys:app:user:review','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001364,9001361,'设置头像','app_user_avatar',3,'','','sys:app:user:avatar','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001365,9001361,'查看登录设备','app_session_list',3,'','','sys:app:session:list','',2,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001366,9001361,'撤销登录设备','app_session_revoke',3,'','','sys:app:session:revoke','',3,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001367,9001362,'授予会员','app_membership_grant',3,'','','sys:app:membership:grant','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001368,9001362,'撤销会员','app_membership_revoke',3,'','','sys:app:membership:revoke','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001369,9001308,'上传文件','file_upload',3,'','','sys:file:upload','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001370,9001361,'编辑资料','app_user_edit',3,'','','sys:app:user:edit','',4,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001371,9001360,'意见反馈','app_feedback',2,'/app/feedback','app_feedback','','message',2,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001372,9001371,'处理反馈','app_feedback_reply',3,'','','sys:app:feedback:reply','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001373,0,'租户管理','tenant',1,'/tenant','','','cloud-server',25,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001374,9001373,'租户列表','tenant_list',2,'/tenant/list','tenant_list','','cloud-server',0,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001375,9001374,'创建租户','tenant_create',3,'','','sys:tenant:create','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001376,9001374,'启停租户','tenant_lifecycle',3,'','','sys:tenant:lifecycle','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001377,9001374,'重置管理员密码','tenant_reset',3,'','','sys:tenant:reset','',2,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001378,9001374,'删除租户','tenant_delete',3,'','','sys:tenant:delete','',3,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001400,0,'AI 功能','ai',1,'/ai','','','robot',30,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001401,9001400,'AI 对话','ai_chat',2,'/ai/chat','ai_chat','','message',0,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001402,9001401,'新建会话','ai_conversation_create',3,'','','sys:ai:conversation:create','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001403,9001401,'发送消息','ai_message_send',3,'','','sys:ai:message:send','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001404,9001401,'删除会话','ai_conversation_delete',3,'','','sys:ai:conversation:delete','',2,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001405,9001401,'查看会话','ai_conversation_list',3,'','','sys:ai:conversation:list','',3,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001410,9001400,'工作流管理','ai_workflow',2,'/ai/workflows','ai_workflow','','apartment',1,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001411,9001410,'查看工作流','ai_workflow_list',3,'','','sys:ai:workflow:list','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001412,9001410,'新建工作流','ai_workflow_create',3,'','','sys:ai:workflow:create','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001413,9001410,'编辑工作流','ai_workflow_edit',3,'','','sys:ai:workflow:edit','',2,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001414,9001410,'删除工作流','ai_workflow_delete',3,'','','sys:ai:workflow:delete','',3,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001415,9001410,'运行工作流','ai_workflow_run',3,'','','sys:ai:workflow:run','',4,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001420,9001400,'任务中心','ai_task',2,'/ai/tasks','ai_task','','audit',2,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001421,9001420,'查看任务','ai_task_list',3,'','','sys:ai:task:list','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001422,9001420,'批准任务','ai_task_approve',3,'','','sys:ai:task:approve','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001423,9001420,'拒绝任务','ai_task_reject',3,'','','sys:ai:task:reject','',2,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001430,9001400,'工具目录','ai_tool',2,'/ai/tools','ai_tool','','api',3,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001431,9001430,'查看工具','ai_tool_list',3,'','','sys:ai:tool:list','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001440,9001400,'知识库','ai_kb',2,'/ai/knowledge','ai_kb','','database',4,1,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001441,9001440,'查看知识库','ai_kb_list',3,'','','sys:ai:kb:list','',0,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001442,9001440,'新建知识库','ai_kb_create',3,'','','sys:ai:kb:create','',1,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001443,9001440,'编辑知识库','ai_kb_edit',3,'','','sys:ai:kb:edit','',2,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001444,9001440,'删除知识库','ai_kb_delete',3,'','','sys:ai:kb:delete','',3,0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(9001445,9001420,'取消任务','ai_task_cancel',3,'','','sys:ai:task:cancel','',3,0,1,'2026-11-07 01:25:31','2026-11-07 01:25:31',NULL);
/*!40000 ALTER TABLE `sys_menu` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_menu_lock`
--

DROP TABLE IF EXISTS `sys_menu_lock`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_menu_lock` (
  `id` bigint unsigned NOT NULL COMMENT '固定锁记录，必须为1',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='菜单结构事务锁';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_menu_lock`
--

LOCK TABLES `sys_menu_lock` WRITE;
/*!40000 ALTER TABLE `sys_menu_lock` DISABLE KEYS */;
INSERT INTO `sys_menu_lock` VALUES (1);
/*!40000 ALTER TABLE `sys_menu_lock` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_role`
--

DROP TABLE IF EXISTS `sys_role`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_role` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `name` varchar(64) NOT NULL COMMENT '角色名称',
  `code` varchar(64) NOT NULL COMMENT '角色编码（英文唯一，casbin 主体用）',
  `description` varchar(512) NOT NULL DEFAULT '' COMMENT '描述',
  `data_scope` varchar(32) NOT NULL DEFAULT 'SELF' COMMENT '数据范围：ALL/DEPT_AND_CHILDREN/DEPT/SELF',
  `sort` int NOT NULL DEFAULT '0' COMMENT '排序',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1启用 2停用',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_code` (`code`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=7 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_role`
--

LOCK TABLES `sys_role` WRITE;
/*!40000 ALTER TABLE `sys_role` DISABLE KEYS */;
INSERT INTO `sys_role` VALUES (1,'超级管理员','super_admin','平台全权限；仅供本地开发初始化','ALL',0,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(2,'全部用户查看','scope_all','只读用户和部门，数据范围为全部用户','ALL',10,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(3,'部门及下级查看','scope_dept_children','只读用户和部门，覆盖当前部门及下级','DEPT_AND_CHILDREN',20,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(4,'本部门查看','scope_dept','只读用户和部门，仅当前部门用户','DEPT',30,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(5,'本人查看','scope_self','只读用户和部门，仅本人用户数据','SELF',40,1,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(6,'停用示例角色','disabled_example','停用角色无用户或菜单关联','SELF',99,2,'2026-01-01 00:00:00','2026-01-01 00:00:00',NULL);
/*!40000 ALTER TABLE `sys_role` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_role_menu`
--

DROP TABLE IF EXISTS `sys_role_menu`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_role_menu` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT,
  `role_id` bigint unsigned NOT NULL,
  `menu_id` bigint unsigned NOT NULL,
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_role_menu` (`role_id`,`menu_id`),
  KEY `idx_menu` (`menu_id`)
) ENGINE=InnoDB AUTO_INCREMENT=172 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='角色-菜单';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_role_menu`
--

LOCK TABLES `sys_role_menu` WRITE;
/*!40000 ALTER TABLE `sys_role_menu` DISABLE KEYS */;
INSERT INTO `sys_role_menu` VALUES (74,2,9001301,'2026-01-01 00:00:00'),(75,2,9001300,'2026-01-01 00:00:00'),(76,2,9001302,'2026-01-01 00:00:00'),(77,2,9001305,'2026-01-01 00:00:00'),(78,2,9001310,'2026-01-01 00:00:00'),(79,2,9001339,'2026-01-01 00:00:00'),(80,3,9001301,'2026-01-01 00:00:00'),(81,3,9001300,'2026-01-01 00:00:00'),(82,3,9001302,'2026-01-01 00:00:00'),(83,3,9001305,'2026-01-01 00:00:00'),(84,3,9001310,'2026-01-01 00:00:00'),(85,3,9001339,'2026-01-01 00:00:00'),(86,4,9001301,'2026-01-01 00:00:00'),(87,4,9001300,'2026-01-01 00:00:00'),(88,4,9001302,'2026-01-01 00:00:00'),(89,4,9001305,'2026-01-01 00:00:00'),(90,4,9001310,'2026-01-01 00:00:00'),(91,4,9001339,'2026-01-01 00:00:00'),(92,5,9001301,'2026-01-01 00:00:00'),(93,5,9001300,'2026-01-01 00:00:00'),(94,5,9001302,'2026-01-01 00:00:00'),(95,5,9001305,'2026-01-01 00:00:00'),(96,5,9001310,'2026-01-01 00:00:00'),(97,5,9001339,'2026-01-01 00:00:00'),(98,1,9001300,'2026-11-07 01:25:31'),(99,1,9001301,'2026-11-07 01:25:31'),(100,1,9001302,'2026-11-07 01:25:31'),(101,1,9001303,'2026-11-07 01:25:31'),(102,1,9001304,'2026-11-07 01:25:31'),(103,1,9001305,'2026-11-07 01:25:31'),(104,1,9001306,'2026-11-07 01:25:31'),(105,1,9001307,'2026-11-07 01:25:31'),(106,1,9001308,'2026-11-07 01:25:31'),(107,1,9001309,'2026-11-07 01:25:31'),(108,1,9001310,'2026-11-07 01:25:31'),(109,1,9001320,'2026-11-07 01:25:31'),(110,1,9001321,'2026-11-07 01:25:31'),(111,1,9001322,'2026-11-07 01:25:31'),(112,1,9001323,'2026-11-07 01:25:31'),(113,1,9001324,'2026-11-07 01:25:31'),(114,1,9001325,'2026-11-07 01:25:31'),(115,1,9001326,'2026-11-07 01:25:31'),(116,1,9001327,'2026-11-07 01:25:31'),(117,1,9001328,'2026-11-07 01:25:31'),(118,1,9001329,'2026-11-07 01:25:31'),(119,1,9001330,'2026-11-07 01:25:31'),(120,1,9001331,'2026-11-07 01:25:31'),(121,1,9001332,'2026-11-07 01:25:31'),(122,1,9001333,'2026-11-07 01:25:31'),(123,1,9001334,'2026-11-07 01:25:31'),(124,1,9001335,'2026-11-07 01:25:31'),(125,1,9001336,'2026-11-07 01:25:31'),(126,1,9001337,'2026-11-07 01:25:31'),(127,1,9001338,'2026-11-07 01:25:31'),(128,1,9001339,'2026-11-07 01:25:31'),(129,1,9001360,'2026-11-07 01:25:31'),(130,1,9001361,'2026-11-07 01:25:31'),(131,1,9001362,'2026-11-07 01:25:31'),(132,1,9001363,'2026-11-07 01:25:31'),(133,1,9001364,'2026-11-07 01:25:31'),(134,1,9001365,'2026-11-07 01:25:31'),(135,1,9001366,'2026-11-07 01:25:31'),(136,1,9001367,'2026-11-07 01:25:31'),(137,1,9001368,'2026-11-07 01:25:31'),(138,1,9001369,'2026-11-07 01:25:31'),(139,1,9001370,'2026-11-07 01:25:31'),(140,1,9001371,'2026-11-07 01:25:31'),(141,1,9001372,'2026-11-07 01:25:31'),(142,1,9001373,'2026-11-07 01:25:31'),(143,1,9001374,'2026-11-07 01:25:31'),(144,1,9001375,'2026-11-07 01:25:31'),(145,1,9001376,'2026-11-07 01:25:31'),(146,1,9001377,'2026-11-07 01:25:31'),(147,1,9001378,'2026-11-07 01:25:31'),(148,1,9001400,'2026-11-07 01:25:31'),(149,1,9001401,'2026-11-07 01:25:31'),(150,1,9001402,'2026-11-07 01:25:31'),(151,1,9001403,'2026-11-07 01:25:31'),(152,1,9001404,'2026-11-07 01:25:31'),(153,1,9001405,'2026-11-07 01:25:31'),(154,1,9001410,'2026-11-07 01:25:31'),(155,1,9001411,'2026-11-07 01:25:31'),(156,1,9001412,'2026-11-07 01:25:31'),(157,1,9001413,'2026-11-07 01:25:31'),(158,1,9001414,'2026-11-07 01:25:31'),(159,1,9001415,'2026-11-07 01:25:31'),(160,1,9001420,'2026-11-07 01:25:31'),(161,1,9001421,'2026-11-07 01:25:31'),(162,1,9001422,'2026-11-07 01:25:31'),(163,1,9001423,'2026-11-07 01:25:31'),(164,1,9001430,'2026-11-07 01:25:31'),(165,1,9001431,'2026-11-07 01:25:31'),(166,1,9001440,'2026-11-07 01:25:31'),(167,1,9001441,'2026-11-07 01:25:31'),(168,1,9001442,'2026-11-07 01:25:31'),(169,1,9001443,'2026-11-07 01:25:31'),(170,1,9001444,'2026-11-07 01:25:31'),(171,1,9001445,'2026-11-07 01:25:31');
/*!40000 ALTER TABLE `sys_role_menu` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `sys_user`
--

DROP TABLE IF EXISTS `sys_user`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `sys_user` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `username` varchar(64) NOT NULL COMMENT '登录名',
  `password_hash` varchar(128) NOT NULL COMMENT '密码哈希（bcrypt）',
  `nickname` varchar(64) NOT NULL DEFAULT '' COMMENT '昵称',
  `real_name` varchar(64) NOT NULL DEFAULT '' COMMENT '真实姓名',
  `email` varchar(128) NOT NULL DEFAULT '' COMMENT '邮箱',
  `phone` varchar(20) NOT NULL DEFAULT '' COMMENT '手机号',
  `dept_id` bigint unsigned DEFAULT NULL COMMENT '所属部门',
  `avatar_file_id` bigint unsigned DEFAULT NULL COMMENT '头像文件ID → sys_file',
  `gender` tinyint NOT NULL DEFAULT '0' COMMENT '性别：0未知 1男 2女',
  `status` tinyint NOT NULL DEFAULT '1' COMMENT '状态：1启用 2停用',
  `remark` varchar(512) NOT NULL DEFAULT '' COMMENT '备注',
  `last_login_at` datetime DEFAULT NULL COMMENT '最后登录时间',
  `last_login_ip` varchar(64) NOT NULL DEFAULT '' COMMENT '最后登录IP',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_username` (`username`),
  KEY `idx_dept` (`dept_id`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB AUTO_INCREMENT=9 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='后台用户';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `sys_user`
--

LOCK TABLES `sys_user` WRITE;
/*!40000 ALTER TABLE `sys_user` DISABLE KEYS */;
INSERT INTO `sys_user` VALUES (1,'admin','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','超级管理员','','','',1,NULL,0,1,'本地开发默认账号，部署前请修改密码','2026-11-07 11:42:27','172.20.0.1','2026-01-01 00:00:00','2026-11-07 11:42:26',NULL),(2,'demo_all','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','全局查看员','演示成员甲','all@example.test','',2,NULL,0,1,'ALL 数据范围示例，只读业务列表','2026-11-07 00:25:26','172.20.0.1','2026-01-01 00:00:00','2026-11-07 00:25:26',NULL),(3,'demo_tree','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','研发负责人','演示成员乙','tree@example.test','',2,NULL,0,1,'研发中心为本部门及下级；切换运营中心时仅本人',NULL,'','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(4,'demo_dept','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','平台负责人','演示成员丙','dept@example.test','',3,NULL,0,1,'DEPT 数据范围示例',NULL,'','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(5,'demo_self','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','平台成员','演示成员丁','self@example.test','',3,NULL,0,1,'SELF 数据范围示例',NULL,'','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(6,'demo_quality','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','质量成员','演示成员戊','quality@example.test','',4,NULL,0,1,'研发下级部门成员，用于比较查询范围',NULL,'','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(7,'demo_operations','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','运营负责人','演示成员己','operations@example.test','',5,NULL,0,1,'其他部门成员，用于比较查询范围',NULL,'','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL),(8,'demo_disabled','$2a$10$5HrvkHotuA/EHstDstu9CO5IroSRQXjytLNM4sI095/NKC0rWo8T6','停用示例用户','演示成员庚','disabled@example.test','',5,NULL,0,2,'停用账号不可登录',NULL,'','2026-01-01 00:00:00','2026-01-01 00:00:00',NULL);
/*!40000 ALTER TABLE `sys_user` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Table structure for table `tenant_registry`
--

DROP TABLE IF EXISTS `tenant_registry`;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!50503 SET character_set_client = utf8mb4 */;
CREATE TABLE `tenant_registry` (
  `id` bigint unsigned NOT NULL AUTO_INCREMENT COMMENT '主键',
  `slug` varchar(32) NOT NULL COMMENT '租户标识（英文小写，子域名前缀）',
  `domain` varchar(128) NOT NULL COMMENT '完整域名，如 t1.localhost',
  `compose_project` varchar(64) NOT NULL COMMENT 'compose 项目名，如 gozero-t-t1',
  `status` varchar(16) NOT NULL DEFAULT 'active' COMMENT '状态：active/stopped/error',
  `mysql_root_password` varchar(128) NOT NULL COMMENT '该租户 MySQL root 密码（随机生成）',
  `jwt_access_secret` varchar(128) NOT NULL COMMENT '该租户后台 JWT 密钥（随机生成）',
  `app_jwt_access_secret` varchar(128) NOT NULL COMMENT '该租户 App JWT 密钥（随机生成）',
  `admin_account_ready` tinyint NOT NULL DEFAULT '0' COMMENT '初始管理员是否就绪',
  `error_msg` varchar(512) NOT NULL DEFAULT '' COMMENT '最近一次错误信息',
  `current_step` varchar(32) DEFAULT '' COMMENT 'provisioning 当前步骤',
  `step_updated_at` datetime DEFAULT NULL COMMENT '步骤更新时间',
  `created_by` bigint unsigned DEFAULT NULL COMMENT '创建操作员ID',
  `created_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at` datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  `deleted_at` datetime DEFAULT NULL COMMENT '软删除（注销租户）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_slug` (`slug`),
  UNIQUE KEY `uk_domain` (`domain`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='租户注册表';
/*!40101 SET character_set_client = @saved_cs_client */;

--
-- Dumping data for table `tenant_registry`
--

LOCK TABLES `tenant_registry` WRITE;
/*!40000 ALTER TABLE `tenant_registry` DISABLE KEYS */;
/*!40000 ALTER TABLE `tenant_registry` ENABLE KEYS */;
UNLOCK TABLES;

--
-- Dumping events for database 'gozero_admin'
--

--
-- Dumping routines for database 'gozero_admin'
--
/*!40103 SET TIME_ZONE=@OLD_TIME_ZONE */;

/*!40101 SET SQL_MODE=@OLD_SQL_MODE */;
/*!40014 SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS */;
/*!40014 SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS */;
/*!40101 SET CHARACTER_SET_CLIENT=@OLD_CHARACTER_SET_CLIENT */;
/*!40101 SET CHARACTER_SET_RESULTS=@OLD_CHARACTER_SET_RESULTS */;
/*!40101 SET COLLATION_CONNECTION=@OLD_COLLATION_CONNECTION */;
/*!40111 SET SQL_NOTES=@OLD_SQL_NOTES */;

-- Dump completed on 2026-11-07 11:42:43
