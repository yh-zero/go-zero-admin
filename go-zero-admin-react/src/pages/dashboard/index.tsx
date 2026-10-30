import { Card, Statistic, Row, Col } from 'antd';

export default function DashboardPage() {
  return (
    <Row gutter={16}>
      <Col span={8}>
        <Card><Statistic title="用户总数" value={0} /></Card>
      </Col>
      <Col span={8}>
        <Card><Statistic title="今日登录" value={0} /></Card>
      </Col>
      <Col span={8}>
        <Card><Statistic title="审计日志" value={0} /></Card>
      </Col>
    </Row>
  );
}
