// Design ref: ux-design.pen Screen C14-D (aJSKl) · C14-M (JUEUD)
import { createFileRoute } from '@tanstack/react-router';
import { Gauge } from 'lucide-react';
import { SettingsPlaceholder } from '../../components/settings/SettingsPlaceholder';

export const Route = createFileRoute('/settings/performance')({
  component: PerformanceSettingsPage,
});

// The one settings page without SettingsPageHeader: C14's page header is
// switched off (GQoB1 / O5N9J enabled:false). The tab is aria-disabled, so the
// only way here is a typed URL, and the placeholder's own title is the whole
// message — a page heading above it would say 效能監控 twice.
function PerformanceSettingsPage() {
  return <SettingsPlaceholder icon={Gauge} title="效能監控" description="查看系統效能指標與趨勢" />;
}
