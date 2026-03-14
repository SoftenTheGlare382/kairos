import React, { useState, useEffect, useRef } from 'react';
import { useParams, Link } from 'react-router-dom';
import { videoApi, likeApi, commentApi, socialApi } from '../api/client';
import { useAuth } from '../context/AuthContext';
import { Heart, Star, MessageSquare, Share2, Play, Pause, Volume2, VolumeX, Maximize } from 'lucide-react';
import { motion, AnimatePresence } from 'framer-motion';
import { clsx } from 'clsx';

const VideoDetail = () => {
  const { id } = useParams();
  const { user } = useAuth();
  const [video, setVideo] = useState(null);
  const [comments, setComments] = useState([]);
  const [newComment, setNewComment] = useState('');
  const [isLiked, setIsLiked] = useState(false);
  const [isFavorited, setIsFavorited] = useState(false);
  const [isPlaying, setIsPlaying] = useState(false);
  const [isMuted, setIsMuted] = useState(false);
  const [progress, setProgress] = useState(0);
  const [duration, setDuration] = useState(0);
  const videoRef = useRef(null);
  const playerRef = useRef(null);

  useEffect(() => {
    const fetchVideo = async () => {
      try {
        const res = await videoApi.getDetail(id);
        setVideo(res.data);
        
        // Record play
        videoApi.recordPlay(id);

        // Check status
        if (user) {
          const likedRes = await likeApi.isLiked(id);
          setIsLiked(likedRes.data.is_liked);
          
          const favRes = await videoApi.isFavorited(id);
          setIsFavorited(favRes.data.is_favorited); // Assuming API returns is_favorited
        }

        // Load comments
        const commentsRes = await commentApi.listAll(id);
        setComments(commentsRes.data || []);
      } catch (err) {
        console.error(err);
      }
    };
    fetchVideo();
  }, [id, user]);

  const togglePlay = () => {
    if (videoRef.current) {
      if (isPlaying) videoRef.current.pause();
      else videoRef.current.play();
      setIsPlaying(!isPlaying);
    }
  };

  const handleTimeUpdate = () => {
    if (videoRef.current) {
      const current = videoRef.current.currentTime;
      const total = videoRef.current.duration;
      setProgress((current / total) * 100);
    }
  };

  const handleLoadedMetadata = () => {
    if (videoRef.current) {
      setDuration(videoRef.current.duration);
    }
  };

  const handleSeek = (e) => {
    const seekTime = (e.target.value / 100) * duration;
    if (videoRef.current) {
      videoRef.current.currentTime = seekTime;
      setProgress(e.target.value);
    }
  };

  const toggleFullScreen = () => {
    if (playerRef.current) {
      if (document.fullscreenElement) {
        document.exitFullscreen();
      } else {
        playerRef.current.requestFullscreen();
      }
    }
  };

  if (!video) return <div className="text-center py-20">Loading...</div>;

  return (
    <div className="flex flex-col lg:flex-row h-[calc(100vh-100px)] max-w-7xl mx-auto gap-8">
      {/* Video Player Section */}
      <div 
        ref={playerRef}
        className="flex-1 bg-black relative group flex items-center justify-center rounded-xl overflow-hidden border border-border shadow-2xl"
      >
        <video
          ref={videoRef}
          src={video.play_url}
          poster={video.cover_url}
          className="max-h-full max-w-full object-contain"
          loop
          onClick={togglePlay}
          onTimeUpdate={handleTimeUpdate}
          onLoadedMetadata={handleLoadedMetadata}
        />
        
        {/* Controls Overlay */}
        <div className="absolute inset-0 bg-gradient-to-t from-black/60 via-transparent to-transparent opacity-0 group-hover:opacity-100 transition-opacity duration-300 flex flex-col justify-end p-6">
          {/* Progress Bar */}
          <div className="mb-4 flex items-center gap-2">
            <span className="text-xs text-white w-10 text-right">
              {videoRef.current ? new Date(videoRef.current.currentTime * 1000).toISOString().substr(14, 5) : "00:00"}
            </span>
            <input
              type="range"
              min="0"
              max="100"
              value={progress}
              onChange={handleSeek}
              className="flex-1 h-1 bg-white/30 rounded-lg appearance-none cursor-pointer [&::-webkit-slider-thumb]:appearance-none [&::-webkit-slider-thumb]:w-3 [&::-webkit-slider-thumb]:h-3 [&::-webkit-slider-thumb]:bg-white [&::-webkit-slider-thumb]:rounded-full"
            />
            <span className="text-xs text-white w-10">
              {videoRef.current ? new Date(videoRef.current.duration * 1000).toISOString().substr(14, 5) : "00:00"}
            </span>
          </div>

          <div className="flex items-center justify-between text-white">
            <button onClick={togglePlay} className="p-2 hover:bg-white/20 rounded-full transition-colors">
              {isPlaying ? <Pause size={32} /> : <Play size={32} />}
            </button>
            
            <div className="flex items-center gap-4">
              <button onClick={toggleMute} className="p-2 hover:bg-white/20 rounded-full transition-colors">
                {isMuted ? <VolumeX size={24} /> : <Volume2 size={24} />}
              </button>
              <button onClick={toggleFullScreen} className="p-2 hover:bg-white/20 rounded-full transition-colors">
                <Maximize size={24} />
              </button>
            </div>
          </div>
        </div>
      </div>

      {/* Info & Comments Section */}
      <div className="w-full lg:w-[400px] flex flex-col bg-surface border-l border-border h-full overflow-hidden">
        {/* Author Info */}
        <div className="p-6 border-b border-border">
          <div className="flex items-center justify-between mb-4">
            <Link to={`/profile/${video.author_id}`} className="flex items-center gap-3 group">
              <div className="w-12 h-12 rounded-full bg-surfaceHighlight flex items-center justify-center text-textSecondary group-hover:bg-white group-hover:text-black transition-colors">
                <span className="font-bold text-lg">{video.username?.[0]?.toUpperCase()}</span>
              </div>
              <div>
                <h3 className="font-bold text-lg group-hover:text-accent transition-colors">{video.username}</h3>
                <p className="text-xs text-textSecondary">Author</p>
              </div>
            </Link>
            <button className="px-4 py-1.5 rounded-full bg-white text-black text-sm font-bold hover:bg-gray-200 transition-colors">
              Follow
            </button>
          </div>
          
          <h1 className="text-xl font-bold mb-2">{video.title}</h1>
          <p className="text-sm text-textSecondary line-clamp-3 mb-4">{video.description}</p>
          
          <div className="flex items-center justify-around py-4 border-t border-border">
            <button onClick={handleLike} className="flex flex-col items-center gap-1 group">
              <div className={clsx("p-3 rounded-full transition-colors", isLiked ? "bg-red-500/20 text-red-500" : "bg-surfaceHighlight group-hover:bg-surfaceHighlight/80")}>
                <Heart size={24} fill={isLiked ? "currentColor" : "none"} />
              </div>
              <span className="text-xs font-medium">{video.likes_count}</span>
            </button>
            
            <button onClick={handleFavorite} className="flex flex-col items-center gap-1 group">
              <div className={clsx("p-3 rounded-full transition-colors", isFavorited ? "bg-yellow-500/20 text-yellow-500" : "bg-surfaceHighlight group-hover:bg-surfaceHighlight/80")}>
                <Star size={24} fill={isFavorited ? "currentColor" : "none"} />
              </div>
              <span className="text-xs font-medium">{video.favorites_count}</span>
            </button>
            
            <button className="flex flex-col items-center gap-1 group">
              <div className="p-3 rounded-full bg-surfaceHighlight group-hover:bg-surfaceHighlight/80 transition-colors">
                <Share2 size={24} />
              </div>
              <span className="text-xs font-medium">Share</span>
            </button>
          </div>
        </div>

        {/* Comments */}
        <div className="flex-1 overflow-y-auto p-6 space-y-6">
          <h3 className="font-bold mb-4 flex items-center gap-2">
            <MessageSquare size={18} />
            Comments ({comments.length})
          </h3>
          
          {comments.map((comment) => (
            <div key={comment.id} className="flex gap-3">
              <div className="w-8 h-8 rounded-full bg-surfaceHighlight flex-shrink-0" />
              <div>
                <p className="text-sm font-bold text-textSecondary mb-1">{comment.username}</p>
                <p className="text-sm">{comment.content}</p>
                <p className="text-xs text-textSecondary mt-1">{new Date(comment.created_at).toLocaleDateString()}</p>
              </div>
            </div>
          ))}
        </div>

        {/* Comment Input */}
        <form onSubmit={handleComment} className="p-4 border-t border-border bg-surfaceHighlight/30">
          <div className="relative">
            <input
              type="text"
              value={newComment}
              onChange={(e) => setNewComment(e.target.value)}
              placeholder="Add a comment..."
              className="w-full pl-4 pr-12 py-3 bg-background rounded-lg text-sm focus:outline-none focus:ring-1 focus:ring-accent transition-all"
            />
            <button 
              type="submit"
              disabled={!newComment.trim()}
              className="absolute right-2 top-1/2 -translate-y-1/2 p-1.5 text-accent hover:text-white disabled:opacity-50 transition-colors"
            >
              Send
            </button>
          </div>
        </form>
      </div>
    </div>
  );
};

export default VideoDetail;
