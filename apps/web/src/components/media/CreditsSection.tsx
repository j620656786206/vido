import { getImageUrl } from '../../lib/image';
import type { CastMember, CrewMember, Creator } from '../../types/tmdb';
import { cn } from '../../lib/utils';

export interface CreditsSectionProps {
  director?: CrewMember;
  cast?: CastMember[];
  createdBy?: Creator[];
  className?: string;
}

/**
 * CreditsSection - Displays cast and crew information
 * AC #1, #2: Director/creator and cast display
 */
export function CreditsSection({ director, cast, createdBy, className }: CreditsSectionProps) {
  // Task 5.3: Show top 6 cast members
  const displayCast = cast?.slice(0, 6) || [];

  if (!director && !createdBy?.length && !displayCast.length) {
    return null;
  }

  return (
    <div className={cn('mt-6', className)} data-testid="credits-section">
      {/* Task 5.2: Director (for movies) or Created by (for TV) */}
      {director && (
        <div className="mb-4">
          <h3 className="mb-2 text-sm font-semibold text-gray-400">導演</h3>
          <CreditPerson
            name={director.name}
            subtitle={director.department}
            profilePath={director.profile_path}
          />
        </div>
      )}

      {createdBy && createdBy.length > 0 && (
        <div className="mb-4">
          <h3 className="mb-2 text-sm font-semibold text-gray-400">創作者</h3>
          <div className="flex flex-wrap gap-3">
            {createdBy.map((creator) => (
              <CreditPerson
                key={creator.id}
                name={creator.name}
                profilePath={creator.profile_path}
              />
            ))}
          </div>
        </div>
      )}

      {/* Task 5.3, 5.4, 5.5: Cast members with profile pictures and character names */}
      {displayCast.length > 0 && (
        <div>
          <h3 className="mb-3 text-sm font-semibold text-gray-400">演員陣容</h3>
          <div className="grid grid-cols-3 gap-3" data-testid="cast-grid">
            {displayCast.map((member) => (
              <CreditPerson
                key={member.id}
                name={member.name}
                subtitle={member.character}
                profilePath={member.profile_path}
              />
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

interface CreditPersonProps {
  name: string;
  subtitle?: string;
  profilePath: string | null;
}

/**
 * Single person credit display with profile image
 */
function CreditPerson({ name, subtitle, profilePath }: CreditPersonProps) {
  // Task 5.5: Handle missing profile images gracefully
  const profileUrl = getImageUrl(profilePath, 'w185');

  return (
    <div className="flex flex-col items-center text-center" data-testid="credit-person">
      {/* Profile image or fallback */}
      <div className="mb-2 h-16 w-16 overflow-hidden rounded-full bg-slate-700">
        {profileUrl ? (
          <img src={profileUrl} alt={name} className="h-full w-full object-cover" loading="lazy" />
        ) : (
          <div className="flex h-full w-full items-center justify-center text-2xl text-slate-500">
            👤
          </div>
        )}
      </div>
      {/* Task 5.4: Actor name and character name */}
      <p className="text-xs font-medium text-white truncate w-full" title={name}>
        {name}
      </p>
      {subtitle && (
        <p className="text-xs text-gray-400 truncate w-full" title={subtitle}>
          {subtitle}
        </p>
      )}
    </div>
  );
}

export default CreditsSection;
