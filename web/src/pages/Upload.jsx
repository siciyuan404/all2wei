import { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import { uploadMaterial } from '../api/material';

function Upload() {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [videoFile, setVideoFile] = useState(null);
  const [subtitleFile, setSubtitleFile] = useState(null);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState('');
  const navigate = useNavigate();

  const handleVideoChange = (e) => {
    const file = e.target.files[0];
    if (file) {
      setVideoFile(file);
      if (!title) {
        setTitle(file.name.replace(/\.[^/.]+$/, ''));
      }
    }
  };

  const handleSubtitleChange = (e) => {
    setSubtitleFile(e.target.files[0]);
  };

  const handleSubmit = async (e) => {
    e.preventDefault();

    if (!videoFile) {
      setError('请选择视频文件');
      return;
    }

    if (!title.trim()) {
      setError('请输入标题');
      return;
    }

    const formData = new FormData();
    formData.append('title', title);
    formData.append('description', description);
    formData.append('video', videoFile);
    if (subtitleFile) {
      formData.append('subtitle', subtitleFile);
    }

    setUploading(true);
    setError('');

    try {
      await uploadMaterial(formData);
      navigate('/');
    } catch (err) {
      setError(err.response?.data?.error || '上传失败');
    } finally {
      setUploading(false);
    }
  };

  return (
    <div className="container">
      <header className="header">
        <h1>上传学习资料</h1>
        <Link to="/" className="btn-secondary">
          返回列表
        </Link>
      </header>

      <div className="upload-form-container">
        {error && <div className="error-message">{error}</div>}

        <form onSubmit={handleSubmit} className="upload-form">
          <div className="form-group">
            <label>视频文件 *</label>
            <input
              type="file"
              accept="video/*"
              onChange={handleVideoChange}
              required
            />
            {videoFile && (
              <span className="file-info">已选择: {videoFile.name}</span>
            )}
          </div>

          <div className="form-group">
            <label>字幕文件 (可选，支持 SRT/VTT)</label>
            <input
              type="file"
              accept=".srt,.vtt,.txt"
              onChange={handleSubtitleChange}
            />
            {subtitleFile && (
              <span className="file-info">已选择: {subtitleFile.name}</span>
            )}
          </div>

          <div className="form-group">
            <label>标题 *</label>
            <input
              type="text"
              value={title}
              onChange={(e) => setTitle(e.target.value)}
              placeholder="输入资料标题"
              required
            />
          </div>

          <div className="form-group">
            <label>描述</label>
            <textarea
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="输入资料描述（可选）"
              rows={4}
            />
          </div>

          <div className="form-actions">
            <button
              type="submit"
              className="btn-primary"
              disabled={uploading}
            >
              {uploading ? '上传中...' : '上传资料'}
            </button>
            <Link to="/" className="btn-secondary">
              取消
            </Link>
          </div>
        </form>
      </div>
    </div>
  );
}

export default Upload;
