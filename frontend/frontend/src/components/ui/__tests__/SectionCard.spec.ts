import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'

import SectionCard from '../SectionCard.vue'

describe('SectionCard', () => {
  it('renders actions slot in the card header', () => {
    const wrapper = mount(SectionCard, {
      props: {
        title: '范围规则',
      },
      slots: {
        actions: '<button class="add-rule">新增规则</button>',
        default: '<div>content</div>',
      },
      global: {
        stubs: {
          ElCard: {
            template: '<div class="el-card"><div class="el-card__header"><slot name="header" /></div><div class="el-card__body"><slot /></div></div>',
          },
        },
      },
    })

    expect(wrapper.text()).toContain('范围规则')
    expect(wrapper.find('.add-rule').exists()).toBe(true)
  })
})
