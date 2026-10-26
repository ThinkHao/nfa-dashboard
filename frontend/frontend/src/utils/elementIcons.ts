import type { Component } from 'vue'
import {
  Avatar,
  Box,
  CircleCheck,
  Coin,
  Connection,
  DataAnalysis,
  Document,
  Filter,
  Histogram,
  House,
  Lock,
  Menu,
  Notebook,
  Refresh,
  School,
  Setting,
  Share,
  TrendCharts,
  User,
  UserFilled,
  Wallet,
} from '@element-plus/icons-vue'

const icons: Record<string, Component> = {
  Avatar,
  Box,
  CircleCheck,
  Coin,
  Connection,
  DataAnalysis,
  Document,
  Filter,
  Histogram,
  House,
  Lock,
  Menu,
  Notebook,
  Refresh,
  School,
  Setting,
  Share,
  TrendCharts,
  User,
  UserFilled,
  Wallet,
}

export function resolveElementIcon(name?: string): Component {
  return (name && icons[name]) || Menu
}
