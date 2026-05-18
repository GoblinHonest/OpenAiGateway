import { RefreshCw } from "lucide-react";
import { SidebarTrigger } from "@/components/ui/sidebar";
import { Button } from "@/components/ui/button";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip";

interface SiteHeaderProps {
  /** Current page title displayed after the brand breadcrumb */
  title: string;
  /** Called when the user clicks the refresh icon */
  onRefresh?: () => void;
}

export function SiteHeader({ title, onRefresh }: SiteHeaderProps) {
  return (
    <header className="flex h-12 shrink-0 items-center gap-3 border-b border-border/40 bg-sidebar px-4">
      {/* ── Left: sidebar toggle ─────────────────────────────── */}
      <SidebarTrigger className="size-6 text-muted-foreground transition-colors hover:text-foreground" />

      {/* ── Page title ───────────────────────────────────────── */}
      <h1 className="text-sm font-medium text-foreground">{title}</h1>

      {/* ── Spacer ───────────────────────────────────────────── */}
      <div className="flex-1" />

      {/* ── Right: refresh ───────────────────────────────────── */}
      <Tooltip>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon-sm"
            className="size-7 text-muted-foreground transition-colors hover:text-foreground"
            onClick={onRefresh}
            aria-label="刷新页面"
          >
            <RefreshCw className="size-3.5" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>刷新</TooltipContent>
      </Tooltip>
    </header>
  );
}
