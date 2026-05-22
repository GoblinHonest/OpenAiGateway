import {
  LayoutDashboard,
  Server,
  Cpu,
  Users,
  Key,
  FileText,
  Settings,
  Activity,
  Sun,
  Moon,
  type LucideIcon,
} from "lucide-react";

import { useThemeValue, type Theme } from "@/hooks/use-theme";
import { cn } from "@/lib/utils";
import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar";

export type Route = "dashboard" | "providers" | "models" | "groups" | "api-keys" | "logs" | "health" | "config";

export const appNavItems: {
  route: Route;
  icon: LucideIcon;
  label: string;
}[] = [
  { route: "dashboard", icon: LayoutDashboard, label: "仪表盘" },
  { route: "providers", icon: Server, label: "服务商" },
  { route: "models", icon: Cpu, label: "模型" },
  { route: "groups", icon: Users, label: "分组" },
  { route: "api-keys", icon: Key, label: "API Keys" },
  { route: "logs", icon: FileText, label: "日志" },
  { route: "health", icon: Activity, label: "健康监控" },
  { route: "config", icon: Settings, label: "配置" },
];

export const SIDEBAR_EXPANDED_WIDTH_PX = 176 * 1.05;
export const SIDEBAR_COLLAPSED_WIDTH_PX = 64;

const navButtonClassName = cn(
  "relative group-data-[state=expanded]:!rounded-xl group-data-[state=expanded]:!px-3 group-data-[state=expanded]:!py-2.5 group-data-[state=expanded]:gap-3",
  "transition-all duration-200 ease-out",
);

interface AppSidebarProps {
  activeRoute: Route;
  onNavigate: (route: Route) => void;
  onThemeChange: (theme: Theme) => void;
}

function ThemeGlyph({ resolved }: { resolved: "light" | "dark" }) {
  if (resolved === "dark") {
    return <Moon className="size-4 shrink-0 transition-transform duration-300 group-hover:rotate-12" strokeWidth={1.75} />;
  }
  return <Sun className="size-4 shrink-0 transition-transform duration-300 group-hover:rotate-12" strokeWidth={1.75} />;
}

export function AppSidebar({
  activeRoute,
  onNavigate,
  onThemeChange,
}: AppSidebarProps) {
  const resolvedTheme = useThemeValue();
  const themeLabel = resolvedTheme === "dark" ? "亮色模式" : "暗色模式";

  return (
    <Sidebar collapsible="icon" variant="sidebar">
      <SidebarHeader className="px-3 pt-3 pb-2">
        <button
          type="button"
          onClick={() => onNavigate("dashboard")}
          className="group/header flex w-full items-center gap-3 rounded-lg px-2 py-2 text-left transition-colors hover:bg-sidebar-accent"
        >
          <div className="relative h-8 w-8 shrink-0">
            <div className="flex h-full w-full items-center justify-center rounded-lg bg-primary">
              <span className="text-sm font-bold text-primary-foreground">G</span>
            </div>
            <span
              aria-hidden
              className="absolute -bottom-0.5 -right-0.5 h-2 w-2 rounded-full bg-emerald-500 ring-2 ring-sidebar"
            />
          </div>
          <div className="flex min-w-0 flex-col leading-tight group-data-[collapsible=icon]/sidebar:hidden">
            <span className="truncate text-sm font-semibold text-sidebar-foreground">
              AI Gateway
            </span>
            <span className="truncate text-[11px] text-sidebar-foreground/50">
              聚合网关
            </span>
          </div>
        </button>
      </SidebarHeader>

      <SidebarContent className="flex-1 px-3 py-2">
        <SidebarMenu className="gap-1">
          {appNavItems.map(({ route, icon: Icon, label }) => {
            const isActive = activeRoute === route;
            return (
              <SidebarMenuItem key={route}>
                <SidebarMenuButton
                  isActive={isActive}
                  tooltip={label}
                  className={cn(
                    "relative gap-3 rounded-lg px-3 py-2 transition-colors",
                    isActive
                      ? "bg-sidebar-accent text-sidebar-foreground font-medium"
                      : "text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground",
                  )}
                  onClick={() => onNavigate(route)}
                >
                  <Icon
                    strokeWidth={1.75}
                    className={cn(
                      "size-4 shrink-0",
                      isActive ? "text-sidebar-foreground" : "text-sidebar-foreground/60",
                    )}
                  />
                  <span className="truncate text-[13px] group-data-[collapsible=icon]/sidebar:hidden">
                    {label}
                  </span>
                </SidebarMenuButton>
              </SidebarMenuItem>
            );
          })}
        </SidebarMenu>
      </SidebarContent>

      <SidebarFooter className="border-t border-sidebar-border/40 px-3 py-3">
        <SidebarMenu className="gap-1">
          <SidebarMenuItem>
            <SidebarMenuButton
              tooltip={themeLabel}
              className="gap-3 rounded-lg px-3 py-2 text-sidebar-foreground/70 transition-colors hover:bg-sidebar-accent/50 hover:text-sidebar-foreground"
              onClick={() => onThemeChange(resolvedTheme === "dark" ? "light" : "dark")}
            >
              <ThemeGlyph resolved={resolvedTheme} />
              <span className="truncate text-[13px] group-data-[collapsible=icon]/sidebar:hidden">
                {themeLabel}
              </span>
            </SidebarMenuButton>
          </SidebarMenuItem>
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  );
}
