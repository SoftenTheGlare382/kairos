import axios from 'axios';

// Create Axios instance
const api = axios.create({
  baseURL: '/', // Use proxy
  timeout: 10000,
});

// Add request interceptor to include token
api.interceptors.request.use(
  (config) => {
    const token = localStorage.getItem('token');
    if (token) {
      config.headers.Authorization = `Bearer ${token}`;
    }
    return config;
  },
  (error) => Promise.reject(error)
);

// Response interceptor to handle 401
api.interceptors.response.use(
  (response) => response,
  (error) => {
    if (error.response && error.response.status === 401) {
      // Optional: Auto logout on 401
      // localStorage.removeItem('token');
      // window.location.href = '/login';
    }
    return Promise.reject(error);
  }
);

// Account API
export const accountApi = {
  register: (username, password) => api.post('/account/register', { username, password }),
  login: (username, password) => api.post('/account/login', { username, password }),
  logout: () => api.post('/account/logout', {}),
  rename: (new_username) => api.post('/account/rename', { new_username }),
  changePassword: (username, old_password, new_password) => api.post('/account/changePassword', { username, old_password, new_password }),
  findByID: (id) => api.post('/account/findByID', { id }),
  findByUsername: (username) => api.post('/account/findByUsername', { username }),
  cancelAccount: () => api.post('/account/cancel', {}),
};

// Feed API
export const feedApi = {
  listLatest: (limit = 20, offset = 0) => api.post('/feed/listLatest', { limit, offset }),
  listByFollowing: (limit = 20, offset = 0) => api.post('/feed/listByFollowing', { limit, offset }),
  listByPopularity: (limit = 20, offset = 0) => api.post('/feed/listByPopularity', { limit, offset }),
};

// Social API
export const socialApi = {
  follow: (following_id) => api.post('/social/follow', { following_id: parseInt(following_id) }),
  unfollow: (following_id) => api.post('/social/unfollow', { following_id: parseInt(following_id) }),
  getFollowers: (user_id, page = 1, page_size = 20) => api.post('/social/followers', { user_id: parseInt(user_id), page, page_size }),
  getFollowing: (user_id, page = 1, page_size = 20) => api.post('/social/following', { user_id: parseInt(user_id), page, page_size }),
};

// IM API
export const imApi = {
  sendMessage: (receiver_id, content) => api.post('/im/send', { receiver_id: parseInt(receiver_id), content }),
  searchMessages: (query, limit = 20, offset = 0) => api.post('/im/search', { query, limit, offset }),
  getConversations: (limit = 20, offset = 0) => api.post('/im/conversations', { limit, offset }),
  markAsRead: (conversation_id) => api.post('/im/read', { conversation_id: parseInt(conversation_id) }),
  getMessages: (conversation_id, limit = 20, offset = 0) => api.post('/im/messages', { conversation_id: parseInt(conversation_id), limit, offset }),
};

// Video API
export const videoApi = {
  listByAuthor: (author_id) => api.post('/video/listByAuthorID', { author_id: parseInt(author_id) }),
  searchVideos: (query, limit = 20, offset = 0) => api.post('/video/search', { query, limit, offset }),
  getDetail: (id) => api.post('/video/getDetail', { id: parseInt(id) }),
  uploadVideo: (formData) => api.post('/video/uploadVideo', formData, { headers: { 'Content-Type': 'multipart/form-data' } }),
  uploadCover: (formData) => api.post('/video/uploadCover', formData, { headers: { 'Content-Type': 'multipart/form-data' } }),
  publish: (data) => api.post('/video/publish', data), // { title, description, play_url, cover_url }
  deleteVideo: (id) => api.post('/video/delete', { id: parseInt(id) }),
  recordPlay: (video_id) => api.post('/video/recordPlay', { video_id: parseInt(video_id) }),
  listPlayRecords: (video_id, limit = 20, offset = 0) => api.post('/video/listPlayRecords', { video_id: parseInt(video_id), limit, offset }),
  
  favorite: (video_id) => api.post('/video/favorite', { video_id: parseInt(video_id) }),
  unfavorite: (video_id) => api.post('/video/unfavorite', { video_id: parseInt(video_id) }),
  isFavorited: (video_id) => api.post('/video/isFavorited', { video_id: parseInt(video_id) }),
  listMyFavorited: () => api.post('/video/listMyFavoritedVideos', {}),
};

// Like API
export const likeApi = {
  like: (video_id) => api.post('/like/like', { video_id: parseInt(video_id) }),
  unlike: (video_id) => api.post('/like/unlike', { video_id: parseInt(video_id) }),
  isLiked: (video_id) => api.post('/like/isLiked', { video_id: parseInt(video_id) }),
  listMyLiked: () => api.post('/like/listMyLikedVideos', {}),
};

// Comment API
export const commentApi = {
  listAll: (video_id) => api.post('/comment/listAll', { video_id: parseInt(video_id) }),
  publish: (video_id, content) => api.post('/comment/publish', { video_id: parseInt(video_id), content }),
  deleteComment: (comment_id) => api.post('/comment/delete', { comment_id: parseInt(comment_id) }),
};

export default api;
