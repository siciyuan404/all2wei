import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useToast } from '../context/ToastContext';
import { uploadMaterial } from '../api/material';
import { Button, Input } from '../components/common';
import { PageLayout } from '../components/layout';
import './Upload.css';

function Upload() {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [videoFile, setVideoFile] = useState(null);
  const [subtitleFile, setSubtitleFile] = useState(null);
  const [uploading, setUploading] = useState(false);
  const [uploadProgress, setUploadProgress] = useState(0);
  const navigate = useNavigate();
  const toast = useToast();

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
      toast.error('请选择视频文件');
      return;
    }

    if (!title.trim()) {
      toast.error('请输入标题');
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
    setUploadProgress(0);

    try {
      await uploadMaterial(formData, {
        onUploadProgress: (progressEvent) => {
          const percentCompleted = progressEvent.total > 0
            ? Math.round((progressEvent.loaded * 100) / progressEvent.total)
            : 0;
          setUploadProgress(percentCompleted);
        },
      });
      toast.success('上传成功');
      navigate('/');
    } catch (err) {
      toast.error(err.response?.data?.error || '上传失败');
    } finally {
      setUploading(false);
      setUploadProgress(0);
    }
  };

  return (
    <PageLayout title="上传学习资料" showBack backTo="/">
      <div className="upload-page">
        <form onSubmit={handleSubmit} className="upload-form">
          <div className="upload-field">
            <label className="upload-label">
              视频文件 <span className="upload-required">*</span>
            </label>
            <div className="upload-file-input">
              <input
                type="file"
                accept="video/*"
                onChange={handleVideoChange}
                required
              />
              {videoFile && (
                <span className="upload-file-name">{videoFile.name}</span>
              )}
            </div>
          </div>

          <div className="upload-field">
            <label className="upload-label">字幕文件 (可选)</label>
            <div className="upload-file-input">
              <input
                type="file"
                accept=".srt,.vtt,.txt"
                onChange={handleSubtitleChange}
              />
              {subtitleFile && (
                <span className="upload-file-name">{subtitleFile.name}</span>
              )}
            </div>
            <p className="upload-hint">支持 SRT、VTT 格式</p>
          </div>

          <Input
            label="标题"
            value={title}
            onChange={(e) => setTitle(e.target.value)}
            placeholder="输入资料标题"
            required
          />

          <div className="upload-field">
            <label className="upload-label">描述</label>
            <textarea
              className="upload-textarea"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
              placeholder="输入资料描述（可选）"
              rows={4}
            />
          </div>

          {uploading && uploadProgress > 0 && (
            <div className="upload-progress">
              <div 
                className="upload-progress-bar" 
                style={{ width: `${uploadProgress}%` }}
              />
              <span className="upload-progress-text">{uploadProgress}%</span>
            </div>
          )}

          <div className="upload-actions">
            <Button
              type="submit"
              variant="primary"
              size="large"
              loading={uploading}
            >
              上传资料
            </Button>
            <Button
              type="button"
              variant="secondary"
              size="large"
              onClick={() => navigate('/')}
              disabled={uploading}
            >
              取消
            </Button>
          </div>
        </form>
      </div>
    </PageLayout>
  );
}

export default Upload;
