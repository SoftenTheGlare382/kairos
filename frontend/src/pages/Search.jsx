import React, { useState } from 'react';
import { videoApi } from '../api/client';
import VideoCard from '../components/VideoCard';
import { Search as SearchIcon } from 'lucide-react';
import { motion } from 'framer-motion';

const Search = () => {
  const [query, setQuery] = useState('');
  const [videos, setVideos] = useState([]);
  const [loading, setLoading] = useState(false);
  const [searched, setSearched] = useState(false);

  const handleSearch = async (e) => {
    e.preventDefault();
    if (!query.trim()) return;
    
    setLoading(true);
    setSearched(true);
    try {
      const res = await videoApi.searchVideos(query);
      setVideos(res.data.list || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto pt-8">
      <form onSubmit={handleSearch} className="relative mb-12">
        <input
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search for videos..."
          className="w-full px-6 py-4 bg-surface border border-border rounded-full text-lg text-white placeholder-textSecondary focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent transition-all duration-300 shadow-[0_0_20px_rgba(255,255,255,0.05)]"
        />
        <button
          type="submit"
          className="absolute right-4 top-1/2 -translate-y-1/2 p-2 bg-white text-black rounded-full hover:bg-gray-200 transition-colors"
        >
          <SearchIcon size={24} />
        </button>
      </form>

      {loading ? (
        <div className="text-center text-textSecondary">Searching...</div>
      ) : searched && videos.length === 0 ? (
        <div className="text-center text-textSecondary">No results found for "{query}"</div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-6">
          {videos.map((video) => (
            <VideoCard key={video.id} video={video} />
          ))}
        </div>
      )}
    </div>
  );
};

export default Search;
