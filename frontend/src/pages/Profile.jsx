import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { useAuth } from '../context/AuthContext';
import { accountApi, videoApi, socialApi, likeApi } from '../api/client';
import VideoCard from '../components/VideoCard';
import { User, Settings, LogOut } from 'lucide-react';
import { motion } from 'framer-motion';
import { clsx } from 'clsx';

const Profile = () => {
  const { id } = useParams();
  const { user, logout } = useAuth();
  const navigate = useNavigate();
  const [profileUser, setProfileUser] = useState(null);
  const [videos, setVideos] = useState([]);
  const [activeTab, setActiveTab] = useState('videos'); // videos, likes, favorites
  const [loading, setLoading] = useState(true);
  const [isFollowing, setIsFollowing] = useState(false);

  const isSelf = !id || (user && user.id === parseInt(id));
  const userId = id ? parseInt(id) : user?.id;

  useEffect(() => {
    if (!userId) return;

    const fetchProfile = async () => {
      setLoading(true);
      try {
        // 1. Get User Info
        const userRes = await accountApi.findByID(userId);
        setProfileUser(userRes.data);

        // 2. Check Follow Status (if not self)
        if (!isSelf && user) {
          // This is tricky, social API doesn't have "isFollowing".
          // We have to check "following" list of current user or "followers" list of target user.
          // For efficiency, maybe just try to follow/unfollow or assume false initially.
          // Or fetch my following list and check if userId is in it.
          // Let's fetch my following list.
          const followingRes = await socialApi.getFollowing(user.id, 1, 100);
          const isF = followingRes.data.list.some(u => u.id === userId);
          setIsFollowing(isF);
        }

        // 3. Get Videos based on tab
        let videoRes;
        if (activeTab === 'videos') {
          videoRes = await videoApi.listByAuthor(userId);
        } else if (activeTab === 'likes' && isSelf) {
          videoRes = await likeApi.listMyLiked();
        } else if (activeTab === 'favorites' && isSelf) {
          videoRes = await videoApi.listMyFavorited();
        }
        
        setVideos(videoRes?.data || []);
      } catch (err) {
        console.error(err);
      } finally {
        setLoading(false);
      }
    };

    fetchProfile();
  }, [userId, activeTab, isSelf, user]);

  const handleFollow = async () => {
    try {
      if (isFollowing) {
        await socialApi.unfollow(userId);
        setIsFollowing(false);
      } else {
        await socialApi.follow(userId);
        setIsFollowing(true);
      }
    } catch (err) {
      console.error(err);
    }
  };

  if (loading) return <div className="text-center py-10">Loading...</div>;
  if (!profileUser) return <div className="text-center py-10">User not found</div>;

  return (
    <div className="max-w-6xl mx-auto pt-8">
      <div className="flex flex-col md:flex-row items-center gap-8 mb-12">
        <div className="w-32 h-32 rounded-full bg-surfaceHighlight flex items-center justify-center border-4 border-surface shadow-[0_0_20px_rgba(255,255,255,0.1)]">
          <User size={64} className="text-textSecondary" />
        </div>
        
        <div className="text-center md:text-left flex-1">
          <h1 className="text-3xl font-bold mb-2">{profileUser.username}</h1>
          <p className="text-textSecondary mb-4">ID: {profileUser.id}</p>
          
          <div className="flex gap-4 justify-center md:justify-start">
            {!isSelf ? (
              <button
                onClick={handleFollow}
                className={clsx(
                  "px-6 py-2 rounded-full font-medium transition-all duration-300",
                  isFollowing
                    ? "bg-surface border border-border text-textSecondary hover:border-red-500 hover:text-red-500"
                    : "bg-white text-black hover:bg-gray-200"
                )}
              >
                {isFollowing ? 'Unfollow' : 'Follow'}
              </button>
            ) : (
              <button
                disabled
                className="px-6 py-2 rounded-full font-medium bg-surface border border-border text-textSecondary opacity-50 cursor-not-allowed"
              >
                It's You
              </button>
            )}
            {isSelf && (
              <button
                onClick={() => navigate('/settings')} // Placeholder
                className="p-2 rounded-full bg-surface border border-border hover:bg-surfaceHighlight transition-colors"
              >
                <Settings size={20} />
              </button>
            )}
            {!isSelf && (
               <button
                onClick={() => navigate(`/messages?target=${userId}`)}
                className="px-6 py-2 rounded-full bg-surface border border-border hover:bg-surfaceHighlight transition-colors"
              >
                Message
              </button>
            )}
          </div>
        </div>
      </div>

      <div className="border-b border-border mb-8">
        <div className="flex gap-8 justify-center md:justify-start">
          {['videos', ...(isSelf ? ['likes', 'favorites'] : [])].map((tab) => (
            <button
              key={tab}
              onClick={() => setActiveTab(tab)}
              className={clsx(
                "pb-4 text-lg font-medium capitalize relative transition-colors",
                activeTab === tab ? "text-white" : "text-textSecondary hover:text-white"
              )}
            >
              {tab}
              {activeTab === tab && (
                <motion.div
                  layoutId="activeTabProfile"
                  className="absolute bottom-0 left-0 right-0 h-0.5 bg-white"
                />
              )}
            </button>
          ))}
        </div>
      </div>

      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 lg:grid-cols-4 gap-6">
        {videos.map((video) => (
          <VideoCard key={video.id} video={video} />
        ))}
      </div>
      
      {videos.length === 0 && (
        <div className="text-center text-textSecondary py-10">No videos yet.</div>
      )}
    </div>
  );
};

export default Profile;
