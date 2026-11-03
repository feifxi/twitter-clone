'use client';

import { Moon, Sun } from 'lucide-react';
import { useTheme } from 'next-themes';
import { DropdownMenuItem } from '@/components/ui/dropdown-menu';
import { cn } from '@/lib/utils';

/**
 * Menu item that switches between light and dark. Menus mount their items when opened, after
 * hydration, so the stored theme is already known here.
 */
export function ThemeToggle({ className }: { className?: string }) {
  const { resolvedTheme, setTheme } = useTheme();
  const dark = resolvedTheme === 'dark';
  const Icon = dark ? Sun : Moon;

  return (
    <DropdownMenuItem
      // Keep the menu open so the change is visible right away
      onSelect={(e) => {
        e.preventDefault();
        setTheme(dark ? 'light' : 'dark');
      }}
      className={cn('cursor-pointer gap-4 p-3 text-[15px]', className)}
    >
      <Icon className="size-5" />
      {dark ? 'Light mode' : 'Dark mode'}
    </DropdownMenuItem>
  );
}
