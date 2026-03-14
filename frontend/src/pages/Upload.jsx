import React, { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { videoApi } from '../api/client';
import { Upload as UploadIcon, X, Check } from 'lucide-react';
import { motion } from 'framer-motion';

const Upload = () => {
  const [videoFile, setVideoFile] = useState(null);
  const [coverFile, setCoverFile] = useState(null);
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [uploading, setUploading] = useState(false);
  const [progress, setProgress] = useState(0);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const handleFileChange = (e, type) => {
    const file = e.target.files[0];
    if (type === 'video') setVideoFile(file);
    else setCoverFile(file);
  };

  const handleUpload = async (e) => {
    e.preventDefault();
    if (!videoFile || !coverFile || !title) {
      setError('Please fill all required fields');
      return;
    }

    setUploading(true);
    setProgress(0);
    setError('');

    try {
      // 1. Upload Video
      const videoFormData = new FormData();
      videoFormData.append('file', videoFile);
      const videoRes = await videoApi.uploadVideo(videoFormData);
      const playUrl = videoRes.data.play_url;
      setProgress(33);

      // 2. Upload Cover
      const coverFormData = new FormData();
      coverFormData.append('file', coverFile);
      const coverRes = await videoApi.uploadCover(coverFormData);
      const coverUrl = coverRes.data.cover_url;
      setProgress(66);

      // 3. Publish
      await videoApi.publish({
        title,
        description,
        play_url: playUrl,
        cover_url: coverUrl,
      });
      setProgress(100);
      
      setTimeout(() => {
        navigate('/');
      }, 1000);
    } catch (err) {
      console.error(err);
      setError(err.response?.data?.error || 'Upload failed');
      setUploading(false);
    }
  };

  return (
    <div className="max-w-2xl mx-auto pt-8">
      <h2 className="text-3xl font-bold mb-8 text-center tracking-widest">UPLOAD</h2>
      
      {error && (
        <div className="mb-4 p-3 bg-red-900/20 border border-red-800 text-red-400 rounded text-sm text-center">
          {error}
        </div>
      )}

      <form onSubmit={handleUpload} className="space-y-6">
        <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
          {/* Video Upload */}
          <div className="relative group cursor-pointer border-2 border-dashed border-border rounded-xl p-8 flex flex-col items-center justify-center hover:border-accent transition-colors h-48">
            <input
              type="file"
              accept="video/mp4"
              onChange={(e) => handleFileChange(e, 'video')}
              className="absolute inset-0 opacity-0 cursor-pointer"
            />
            {videoFile ? (
              <div className="text-center">
                <Check className="mx-auto mb-2 text-green-500" size={32} />
                <p className="text-sm truncate max-w-[150px]">{videoFile.name}</p>
              </div>
            ) : (
              <div className="text-center text-textSecondary group-hover:text-white transition-colors">
                <UploadIcon className="mx-auto mb-2" size={32} />
                <p className="text-sm">Select Video (.mp4)</p>
              </div>
            )}
          </div>

          {/* Cover Upload */}
          <div className="relative group cursor-pointer border-2 border-dashed border-border rounded-xl p-8 flex flex-col items-center justify-center hover:border-accent transition-colors h-48">
            <input
              type="file"
              accept="image/*"
              onChange={(e) => handleFileChange(e, 'cover')}
              className="absolute inset-0 opacity-0 cursor-pointer"
            />
            {coverFile ? (
              <div className="text-center">
                <Check className="mx-auto mb-2 text-green-500" size={32} />
                <p className="text-sm truncate max-w-[150px]">{coverFile.name}</p>
              </div>
            ) : (
              <div className="text-center text-textSecondary group-hover:text-white transition-colors">
                <UploadIcon className="mx-auto mb-2" size={32} />
                <p className="text-sm">Select Cover</p>
              </div>
            )}
          </div>
        </div>

        <div>
          <label className="block text-sm font-medium text-textSecondary mb-2">Title</label>
          <input
            type="text"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            className="w-full px-4 py-3 bg-surfaceHighlight border border-border rounded-lg focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent transition-all duration-300 text-white placeholder-gray-600"
            placeholder="Give your video a title"
            required
          />
        </div>

        <div>
          <label className="block text-sm font-medium text-textSecondary mb-2">Description</label>
          <textarea
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            className="w-full px-4 py-3 bg-surfaceHighlight border border-border rounded-lg focus:outline-none focus:border-accent focus:ring-1 focus:ring-accent transition-all duration-300 text-white placeholder-gray-600 h-32 resize-none"
            placeholder="Describe your video..."
          />
        </div>

        <button
          type="submit"
          disabled={uploading}
          className="w-full py-4 bg-white text-black font-bold rounded-lg hover:bg-gray-200 transition-colors duration-300 disabled:opacity-50 disabled:cursor-not-allowed shadow-[0_0_20px_rgba(255,255,255,0.2)]"
        >
          {uploading ? `Uploading... ${progress}%` : 'Publish Video'}
        </button>
      </form>
    </div>
  );
};

export default Upload;
