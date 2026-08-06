import { describe, expect, it } from 'vitest'
import type { RouteLocationNormalized, RouteRecordRaw } from 'vue-router'
import { generateMenuItems } from './useMenu'

const route = (
  name: string,
  meta: Record<string, unknown> = {},
  children: RouteRecordRaw[] = [],
): RouteRecordRaw => ({
  path: `/${name}`,
  name,
  component: { render: () => null },
  meta,
  children,
} as unknown as RouteRecordRaw)

const permission = (isAdmin: boolean) => ({
  accessRouter: (item: RouteLocationNormalized | RouteRecordRaw) =>
    !item.meta?.requiresAdmin || isAdmin,
})

describe('generateMenuItems', () => {
  it('filters admin routes by permission', () => {
    const routes = [
      route('public', { title: '公开' }),
      route('admin', { title: '管理', requiresAdmin: true }),
    ]

    expect(generateMenuItems(routes, permission(false), false).map((item) => item.name))
      .toEqual(['public'])
    expect(generateMenuItems(routes, permission(true), false).map((item) => item.name))
      .toEqual(['public', 'admin'])
  })

  it('skips hidden and compact-hidden menu entries', () => {
    const routes = [
      route('public', { title: '公开' }),
      route('hidden', { title: '隐藏', hideInMenu: true }),
      route('compact', { title: '紧凑隐藏', hideOnCompactNavigation: true }),
    ]

    expect(generateMenuItems(routes, permission(true), true).map((item) => item.name))
      .toEqual(['public'])
  })

  it('recurses into children and carries parent names', () => {
    const routes = [
      route('group', { title: '分组' }, [
        route('child', { title: '子项' }),
      ]),
    ]

    const menu = generateMenuItems(routes, permission(true), false)
    expect(menu[0]?.children?.[0]).toMatchObject({
      name: 'child',
      title: '子项',
      parentNames: ['group'],
    })
  })
})
