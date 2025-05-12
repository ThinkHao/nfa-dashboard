# Grafana 时间序列数据展示方案

## 概述

Grafana 是一个强大的开源数据可视化和监控平台，特别擅长处理时间序列数据。它能够在保持时间颗粒度对齐的同时，提供高性能的数据展示。本文档将详细介绍 Grafana 在时间序列数据展示方面的技术和方案，以及如何应用这些技术到我们的流速图展示中。

## Grafana 的核心技术

### 1. 数据降采样和聚合

Grafana 使用多种技术来处理大量时间序列数据：

- **自动降采样**：根据显示区域的像素宽度自动调整数据点数量，避免过度绘制
- **智能聚合**：对于长时间范围的查询，自动应用聚合函数（如平均值、最大值、最小值等）
- **动态分辨率**：根据时间范围自动调整查询的时间分辨率

### 2. 查询优化

- **查询分片**：将长时间范围的查询分解为多个较小的查询，并行处理
- **查询缓存**：缓存常用时间范围的查询结果
- **延迟加载**：仅加载当前可见区域的数据，滚动或缩放时动态加载更多数据

### 3. 数据模型和存储

- **时间序列数据库集成**：与专为时间序列数据优化的数据库（如 InfluxDB、Prometheus、TimescaleDB）无缝集成
- **数据压缩**：使用专门的时间序列压缩算法减少存储和传输开销
- **预计算聚合**：预先计算和存储不同时间粒度的聚合数据

### 4. 渲染优化

- **Canvas 渲染**：使用 HTML5 Canvas 进行高效渲染，而不是 DOM 元素
- **WebGL 加速**：对于大量数据点，使用 WebGL 进行硬件加速渲染
- **增量渲染**：只重新渲染发生变化的部分

## 时间粒度对齐策略

Grafana 采用以下策略确保时间粒度对齐：

### 1. 时间对齐算法

- **自动时间对齐**：根据查询时间范围自动选择合适的时间间隔
- **时间边界对齐**：确保数据点对齐到规则的时间边界（如整点、整分钟等）
- **时区处理**：正确处理不同时区和夏令时的影响

### 2. 数据点插值和填充

- **零值填充**：对于缺失的数据点，可以选择填充零值或 null
- **线性插值**：在数据点之间进行线性插值，平滑显示
- **阶梯插值**：使用阶梯函数插值，适合表示状态变化

### 3. 时间粒度自适应

- **动态粒度调整**：根据查询时间范围自动调整时间粒度
- **粒度层级**：支持多级时间粒度，从秒级到年级
- **用户可配置**：允许用户手动指定时间粒度，覆盖自动选择

## 应用到我们的流速图

根据 Grafana 的技术和方案，我们可以对流速图进行以下改进：

### 1. 后端优化

```go
// 根据时间范围动态调整查询策略
func determineQueryStrategy(startTime, endTime time.Time) (string, int) {
    // 计算时间范围（分钟）
    diffMinutes := endTime.Sub(startTime).Minutes()
    
    // 根据时间范围选择合适的粒度
    if diffMinutes <= 360 { // 6小时以内
        return "5m", int(diffMinutes/5) + 10 // 原始5分钟粒度
    } else if diffMinutes <= 1440 { // 24小时以内
        return "15m", int(diffMinutes/15) + 10 // 15分钟粒度
    } else if diffMinutes <= 10080 { // 7天以内
        return "1h", int(diffMinutes/60) + 10 // 1小时粒度
    } else {
        return "1d", int(diffMinutes/1440) + 10 // 1天粒度
    }
}

// 查询不同粒度的数据
func queryTimeSeriesData(db *sql.DB, startTime, endTime time.Time, granularity string) ([]DataPoint, error) {
    var query string
    
    switch granularity {
    case "5m":
        // 查询原始5分钟粒度数据
        query = `SELECT create_time, total_recv, total_send FROM traffic WHERE create_time BETWEEN ? AND ?`
    case "15m", "1h", "1d":
        // 查询聚合数据
        var timeFormat string
        if granularity == "15m" {
            timeFormat = "DATE_FORMAT(create_time, '%Y-%m-%d %H:%i') as time_bucket"
        } else if granularity == "1h" {
            timeFormat = "DATE_FORMAT(create_time, '%Y-%m-%d %H:00') as time_bucket"
        } else {
            timeFormat = "DATE(create_time) as time_bucket"
        }
        
        query = fmt.Sprintf(`
            SELECT 
                %s,
                AVG(total_recv) as total_recv,
                AVG(total_send) as total_send
            FROM traffic
            WHERE create_time BETWEEN ? AND ?
            GROUP BY time_bucket
            ORDER BY time_bucket
        `, timeFormat)
    }
    
    // 执行查询并返回结果
    // ...
}
```

### 2. 前端优化

```javascript
// 根据时间范围和数据点数量动态调整显示策略
function determineDisplayStrategy(timeRange, dataPoints) {
    const rangeHours = (timeRange.end - timeRange.start) / (1000 * 60 * 60);
    const pointDensity = dataPoints.length / rangeHours;
    
    // 如果数据点密度过高，进行客户端降采样
    if (pointDensity > 10 && dataPoints.length > 1000) {
        return {
            needsDownsampling: true,
            samplingFactor: Math.ceil(dataPoints.length / 1000)
        };
    }
    
    return {
        needsDownsampling: false
    };
}

// 客户端降采样
function downsampleData(dataPoints, samplingFactor) {
    // 使用LTTB算法进行降采样，保留数据特征
    return largestTriangleThreeBuckets(dataPoints, Math.floor(dataPoints.length / samplingFactor));
}

// 处理不同粒度的数据展示
function formatTimeLabel(timestamp, granularity) {
    const date = new Date(timestamp);
    
    switch (granularity) {
    case "5m":
        return `${date.getHours()}:${date.getMinutes().toString().padStart(2, '0')}`;
    case "15m":
        return `${date.getHours()}:${date.getMinutes().toString().padStart(2, '0')}`;
    case "1h":
        return `${date.getMonth()+1}/${date.getDate()} ${date.getHours()}:00`;
    case "1d":
        return `${date.getMonth()+1}/${date.getDate()}`;
    }
}
```

### 3. 数据缓存和预加载

```javascript
// 实现数据缓存
const dataCache = new Map();

async function fetchData(startTime, endTime, granularity) {
    const cacheKey = `${startTime.toISOString()}_${endTime.toISOString()}_${granularity}`;
    
    // 检查缓存
    if (dataCache.has(cacheKey)) {
        return dataCache.get(cacheKey);
    }
    
    // 获取数据
    const data = await api.getTrafficData({
        start_time: startTime.toISOString(),
        end_time: endTime.toISOString(),
        granularity: granularity
    });
    
    // 存入缓存
    dataCache.set(cacheKey, data);
    
    return data;
}

// 预加载相邻时间范围的数据
function preloadAdjacentData(currentStartTime, currentEndTime, granularity) {
    const timeRange = currentEndTime - currentStartTime;
    
    // 预加载前一个时间范围
    const prevStartTime = new Date(currentStartTime.getTime() - timeRange);
    const prevEndTime = new Date(currentStartTime);
    fetchData(prevStartTime, prevEndTime, granularity);
    
    // 预加载后一个时间范围
    const nextStartTime = new Date(currentEndTime);
    const nextEndTime = new Date(currentEndTime.getTime() + timeRange);
    fetchData(nextStartTime, nextEndTime, granularity);
}
```

## 总结和建议

基于 Grafana 的技术和方案，我们可以对流速图进行以下改进：

1. **实现多级时间粒度**：
   - 对于短时间范围（6小时以内）：使用原始5分钟粒度数据
   - 对于中等时间范围（6-24小时）：使用15分钟粒度聚合数据
   - 对于长时间范围（1-7天）：使用1小时粒度聚合数据
   - 对于超长时间范围（7天以上）：使用1天粒度聚合数据

2. **优化数据查询**：
   - 实现查询缓存
   - 根据时间范围自动选择合适的查询策略
   - 预计算并存储不同粒度的聚合数据

3. **前端渲染优化**：
   - 实现客户端降采样
   - 使用 Canvas 或 WebGL 进行高效渲染
   - 动态调整时间轴标签格式

4. **用户体验改进**：
   - 显示当前使用的时间粒度
   - 允许用户手动选择时间粒度
   - 提供数据点悬停详情

通过实施这些改进，我们可以在保持时间颗粒度对齐的同时，提供高性能的流速图展示，无论用户选择什么时间范围。
