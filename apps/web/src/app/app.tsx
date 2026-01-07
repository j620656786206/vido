// Uncomment this line to use CSS modules
// import styles from './app.module.css';
import NxWelcome from './nx-welcome';
import type { Movie, ApiResponse } from '@vido/shared-types';

export function App() {
  // Example usage of shared types to verify path alias works
  const exampleMovie: Movie = {
    id: '1',
    title: 'Example Movie',
    releaseDate: '2024-01-01',
    genres: ['Action', 'Adventure'],
    rating: 8.5,
  };

  const exampleResponse: ApiResponse<Movie> = {
    success: true,
    data: exampleMovie,
    message: 'Movie fetched successfully',
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <NxWelcome title="web" />
      {/* Types are working: {exampleResponse.data?.title} */}
    </div>
  );
}

export default App;
