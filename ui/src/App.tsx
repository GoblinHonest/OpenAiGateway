import { Suspense, lazy, useCallback, useEffect, useState, type CSSProperties } from "react";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Toaster } from "@/components/ui/sonner";
import { useTheme } from "@/hooks/use-theme";
import { useRouteTransition } from "@/hooks/use-route-transition";
import { PageStage } from "@/components/layout/page-stage";
import {
  AppSidebar,
  SIDEBAR_COLLAPSED_WIDTH_PX,
  SIDEBAR_EXPANDED_WIDTH_PX,
  type Route,
} from "@/components/layout/sidebar";
import { SiteHeader } from "@/components/layout/site-header";
import { SidebarInset, SidebarProvider } from "@/components/ui/sidebar";
import { Skeleton } from "@/components/ui/skeleton";
import { isAuthenticated } from "@/api/auth";
import Login from "@/components/Login";

const Dashboard = lazy(() => import("@/pages/Dashboard"));
const Providers = lazy(() => import("@/pages/Providers"));
const Models = lazy(() => import("@/pages/Models"));
const Groups = lazy(() => import("@/pages/Groups"));
const ApiKeys = lazy(() => import("@/pages/ApiKeys"));
const Logs = lazy(() => import("@/pages/Logs"));
const Config = lazy(() => import("@/pages/Config"));
const Health = lazy(() => import("@/pages/Health"));

const routeLabels: Record<Route, string> = {
  dashboard: "仪表盘",
  providers: "服务商",
  models: "模型",
  groups: "分组",
  "api-keys": "API Keys",
  logs: "日志",
  health: "健康监控",
  config: "配置",
};

const routeOrder: Route[] = ["dashboard", "providers", "models", "groups", "api-keys", "logs", "health", "config"];

function PageShellSkeleton() {
  return (
    <div className="space-y-6">
      <div className="space-y-2">
        <Skeleton className="h-6 w-32" />
        <Skeleton className="h-4 w-72" />
      </div>
      <div className="rounded-2xl border border-border bg-card p-6">
        <div className="space-y-4">
          {Array.from({ length: 5 }).map((_, index) => (
            <div key={index} className="flex items-center justify-between border-b border-border/60 pb-4 last:border-b-0">
              <div className="space-y-2">
                <Skeleton className="h-4 w-36" />
                <Skeleton className="h-3 w-56" />
              </div>
              <Skeleton className="h-8 w-20 rounded-xl" />
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

export default function App() {
  const [loggedIn, setLoggedIn] = useState(false);
  const [checking, setChecking] = useState(true);
  const [route, setRoute] = useState<Route>("dashboard");
  const { theme, setTheme } = useTheme();
  const [sidebarOpen, setSidebarOpen] = useState(
    () => localStorage.getItem("sidebar_collapsed") === "false"
  );

  useEffect(() => {
    const checkAuth = async () => {
      const token = localStorage.getItem('admin_token');
      if (!token) {
        setLoggedIn(false);
        setChecking(false);
        return;
      }

      // 验证token是否有效
      try {
        const res = await fetch('/admin/v1/dashboard/overview', {
          headers: { Authorization: `Bearer ${token}` },
        });
        setLoggedIn(res.ok);
        if (!res.ok) {
          localStorage.removeItem('admin_token');
        }
      } catch {
        setLoggedIn(false);
        localStorage.removeItem('admin_token');
      }
      setChecking(false);
    };
    checkAuth();
  }, []);

  const handleThemeChange = useCallback((nextTheme: "light" | "dark" | "system") => {
    setTheme(nextTheme);
  }, [setTheme]);

  const routeTransition = useRouteTransition(route, { durationMs: 240 });

  if (checking) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="h-10 w-10 animate-spin rounded-full border-3 border-border border-t-primary" />
      </div>
    );
  }

  if (!loggedIn) {
    return <Login onLogin={() => setLoggedIn(true)} />;
  }

  const renderPage = (targetRoute: Route) => {
    switch (targetRoute) {
      case "dashboard":
        return <Dashboard />;
      case "providers":
        return <Providers />;
      case "models":
        return <Models />;
      case "groups":
        return <Groups />;
      case "api-keys":
        return <ApiKeys />;
      case "logs":
        return <Logs />;
      case "health":
        return <Health />;
      case "config":
        return <Config />;
      default:
        return null;
    }
  };

  return (
    <div className="flex h-screen w-screen overflow-hidden bg-background">
      <TooltipProvider delayDuration={200}>
        <SidebarProvider
          open={sidebarOpen}
          onOpenChange={(open) => {
            setSidebarOpen(open);
            localStorage.setItem("sidebar_collapsed", String(!open));
          }}
          style={
            {
              "--sidebar-width": `${SIDEBAR_EXPANDED_WIDTH_PX}px`,
              "--sidebar-width-icon": `${SIDEBAR_COLLAPSED_WIDTH_PX}px`,
            } as CSSProperties
          }
          className="flex min-h-0 flex-1 overflow-hidden"
        >
          <AppSidebar
            activeRoute={route}
            onNavigate={setRoute}
            onThemeChange={handleThemeChange}
          />
          <SidebarInset className="max-h-screen overflow-hidden">
            <SiteHeader
              title={routeLabels[route]}
              onRefresh={() => window.location.reload()}
            />
            <div className="relative min-h-0 flex-1 overflow-hidden">
              {routeOrder
                .filter((candidate) => routeTransition.mountedRoutes.includes(candidate))
                .map((candidate) => (
                  <PageStage
                    key={candidate}
                    state={routeTransition.getStage(candidate)}
                  >
                    <Suspense fallback={<PageShellSkeleton />}>
                      {renderPage(candidate)}
                    </Suspense>
                  </PageStage>
                ))}
            </div>
          </SidebarInset>
        </SidebarProvider>
      </TooltipProvider>
      <Toaster />
    </div>
  );
}
