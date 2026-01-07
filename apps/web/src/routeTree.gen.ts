import { Route as rootRoute } from './routes/__root';
import { Route as IndexRoute } from './routes/index';

const indexRouteWithChildren = IndexRoute;

export const routeTree = rootRoute.addChildren([indexRouteWithChildren]);
