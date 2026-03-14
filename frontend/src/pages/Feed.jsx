import React, { useState, useEffect } from 'react';
import { feedApi } from '../api/client';
import VideoCard from '../components/VideoCard';
import { motion } from 'framer-motion';
import { clsx } from 'clsx';

const Feed = () => {
  const [activeTab, setActiveTab] = useState('latest'); // latest, following, popular
  const [videos, setVideos] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const fetchVideos = async () => {
    setLoading(true);
    setError('');
    try {
      let res;
      if (activeTab === 'latest') {
        res = await feedApi.listLatest();
      } else if (activeTab === 'following') {
        res = await feedApi.listByFollowing();
      } else if (activeTab === 'popular') {
        res = await feedApi.listByPopularity();
      }
      setVideos(res.data || []);
    } catch (err) {
      console.error(err);
      setError('Failed to load videos');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchVideos();
  }, [activeTab]);

  const tabs = [
    { id: 'latest', label: 'Latest' },
    { id: 'popular', label: 'Popular' },
    { id: 'following', label: 'Following' },
  ];

  return (
    <div className="max-w-7xl mx-auto">
      <div className="flex items-center justify-center mb-8 gap-8">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActiveTab(tab.id)}
            className={clsx(
              "text-lg font-medium pb-2 transition-all duration-300 relative",
              activeTab === tab.id
                ? "text-white"
                : "text-textSecondary hover:text-white"
            )}
          >
            {tab.label}
            {activeTab === tab.id && (
              <motion.div
                layoutId="activeTab"
                className="absolute bottom-0 left-0 right-0 h-0.5 bg-white shadow-[0_0_10px_rgba(255,255,255,0.5)]"
              />
            )}
          </button>
        ))}
      </div>

      {loading ? (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
          {[...Array(8)].map((_, i) => (
            <div key={i} className="aspect-[9/16] bg-surfaceHighlight rounded-xl animate-pulse" />
          ))}
        </div>
      ) : error ? (
        <div className="text-center text-red-500 py-10">{error}</div>
      ) : videos.length === 0 ? (
        <div className="text-center text-textSecondary py-10">No videos found.</div>
      ) : (
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
          {videos.map((video) => (
            <VideoCard key={video.id} video={video} />
          ))}
        </div>
      )}
    </div>
  );
};

export default Feed;
