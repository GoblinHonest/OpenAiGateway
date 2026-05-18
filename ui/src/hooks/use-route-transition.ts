import { useCallback, useEffect, useRef, useState } from "react";

type StageState = "active" | "exiting" | "idle";

interface RouteTransitionResult {
  mountedRoutes: string[];
  getStage: (route: string) => StageState;
}

export function useRouteTransition(
  activeRoute: string,
  options: { durationMs?: number } = {},
): RouteTransitionResult {
  const { durationMs = 240 } = options;
  const [mountedRoutes, setMountedRoutes] = useState<string[]>([activeRoute]);
  const [stages, setStages] = useState<Record<string, StageState>>({ [activeRoute]: "active" });
  const prevRoute = useRef(activeRoute);

  useEffect(() => {
    if (activeRoute === prevRoute.current) return;

    const oldRoute = prevRoute.current;
    prevRoute.current = activeRoute;

    // Start exit animation for old route
    setStages((s) => ({ ...s, [oldRoute]: "exiting", [activeRoute]: "active" }));

    // Mount new route
    setMountedRoutes((routes) =>
      routes.includes(activeRoute) ? routes : [...routes, activeRoute]
    );

    // Unmount old route after animation
    const timer = setTimeout(() => {
      setMountedRoutes((routes) => routes.filter((r) => r !== oldRoute));
      setStages((s) => {
        const next = { ...s };
        delete next[oldRoute];
        return next;
      });
    }, durationMs);

    return () => clearTimeout(timer);
  }, [activeRoute, durationMs]);

  const getStage = useCallback(
    (route: string): StageState => stages[route] ?? "idle",
    [stages]
  );

  return { mountedRoutes, getStage };
}
