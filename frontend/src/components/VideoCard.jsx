import React from 'react';
import { Link } from 'react-router-dom';
import { Heart, Play } from 'lucide-react';
import { motion } from 'framer-motion';

const VideoCard = ({ video }) => {
  return (
    <motion.div
      whileHover={{ y: -5 }}
      className="group relative bg-surface rounded-xl overflow-hidden border border-border transition-all duration-300 hover:shadow-[0_0_20px_rgba(255,255,255,0.1)]"
    >
      <Link to={`/video/${video.id}`} className="block aspect-[9/16] relative overflow-hidden">
        {video.cover_url ? (
          <img
            src={video.cover_url}
            alt={video.title}
            className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-105"
          />
        ) : (
          <div className="w-full h-full bg-surfaceHighlight flex items-center justify-center text-textSecondary">
            <Play size={48} />
          </div>
        )}
        <div className="absolute inset-0 bg-gradient-to-t from-black/80 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300 flex flex-col justify-end p-4">
          <div className="flex items-center gap-2 text-white">
            <Play size={16} fill="white" />
            <span className="text-sm font-medium">Play</span>
          </div>
        </div>
      </Link>
      
      <div className="p-4">
        <h3 className="text-lg font-bold text-white mb-1 truncate">{video.title}</h3>
        <div className="flex items-center justify-between text-sm text-textSecondary">
          <Link to={`/profile/${video.author_id}`} className="hover:text-white transition-colors">
            @{video.username || 'Unknown'}
          </Link>
          <div className="flex items-center gap-1">
            <Heart size={14} className={video.is_liked ? "fill-red-500 text-red-500" : ""} />
            <span>{video.likes_count}</span>
          </div>
        </div>
      </div>
    </motion.div>
  );
};

export default VideoCard;
