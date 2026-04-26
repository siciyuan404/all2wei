import { useEffect, useState, useCallback } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { useToast } from '../context/ToastContext';
import { getMaterials, getFolders, deleteMaterial, syncMaterials } from '../api/material';
import { Button } from '../components/common';
import { PageLayout } from '../components/layout';
import './MaterialList.css';

const RECENT_FOLDERS_KEY = 'recent_folders';
const MAX_RECENT = 10;

function getRecentFolders() {
  try {
    return JSON.parse(localStorage.getItem(RECENT_FOLDERS_KEY) || '[]');
  } catch {
    return [];
  }
}

function addRecentFolder(folder) {
  const recent = getRecentFolders().filter(f => f !== folder);
  recent.unshift(folder);
  localStorage.setItem(RECENT_FOLDERS_KEY, JSON.stringify(recent.slice(0, MAX_RECENT)));
}

function formatFolderName(folder) {
  const parts = folder.split('/');
  return parts[parts.length - 1] || folder;
}

function MaterialList() {
  const [materials, setMaterials] = useState([]);
  const [folders, setFolders] = useState([]);
  const [recentFolders, setRecentFolders] = useState([]);
  const [selectedFolder, setSelectedFolder] = useState(null);
  const [loading, setLoading] = useState(true);
  const [syncing, setSyncing] = useState(false);
  const toast = useToast();
  const navigate = useNavigate();

  useEffect(() => {
    fetchFolders();
    setRecentFolders(getRecentFolders());
  }, []);

  useEffect(() => {
    if (selectedFolder) {
      fetchMaterials(selectedFolder);
    } else {
      setMaterials([]);
      setLoading(false);
    }
  }, [selectedFolder]);

  const fetchFolders = async () => {
    try {
      const response = await getFolders();
      setFolders(response.data || []);
    } catch (err) {
      toast.error('获取文件夹列表失败');
    }
  };

  const fetchMaterials = async (folder) => {
    setLoading(true);
    try {
      const response = await getMaterials(folder);
      setMaterials(response.data || []);
    } catch (err) {
      toast.error('获取资料列表失败');
    } finally {
      setLoading(false);
    }
  };

  const handleFolderClick = (folderName) => {
    setSelectedFolder(folderName);
    addRecentFolder(folderName);
    setRecentFolders(getRecentFolders());
  };

  const handleBackToFolders = () => {
    setSelectedFolder(null);
    setMaterials([]);
  };

  const handleSync = async () => {
    setSyncing(true);
    try {
      const response = await syncMaterials();
      toast.success(response.data.message);
      fetchFolders();
      if (selectedFolder) fetchMaterials(selectedFolder);
    } catch (err) {
      toast.error(err.response?.data?.error || '同步失败');
    } finally {
      setSyncing(false);
    }
  };

  const handleDelete = async (id) => {
    if (!window.confirm('确定要删除这个学习资料吗？')) return;
    try {
      await deleteMaterial(id);
      setMaterials(materials.filter((m) => m.id !== id));
      toast.success('删除成功');
    } catch (err) {
      toast.error('删除失败');
    }
  };

  const handleMaterialClick = (id) => {
    navigate(`/watch/${id}`);
  };

  const actions = (
    <>
      <Link to="/upload">
        <Button variant="primary" size="small">+ 上传</Button>
      </Link>
      <Button
        variant="secondary"
        size="small"
        onClick={handleSync}
        loading={syncing}
      >
        同步 MinIO
      </Button>
    </>
  );

  if (selectedFolder) {
    return (
      <PageLayout title={formatFolderName(selectedFolder)} actions={actions}>
        <div className="folder-breadcrumb">
          <button className="breadcrumb-link" onClick={handleBackToFolders}>
            ← 所有文件夹
          </button>
          <span className="breadcrumb-sep">/</span>
          <span className="breadcrumb-current">{formatFolderName(selectedFolder)}</span>
        </div>
        {loading ? (
          <div className="material-list-loading">加载中...</div>
        ) : materials.length === 0 ? (
          <div className="material-list-empty">该文件夹下暂无视频</div>
        ) : (
          <div className="material-video-list">
            {materials.map((m) => (
              <div
                key={m.id}
                className="material-video-item"
                onClick={() => handleMaterialClick(m.id)}
              >
                <div className="video-item-icon">▶</div>
                <div className="video-item-info">
                  <div className="video-item-title">{m.title}</div>
                  {m.has_subtitle && <div className="video-item-badge">字幕</div>}
                </div>
                <button
                  className="video-item-delete"
                  onClick={(e) => { e.stopPropagation(); handleDelete(m.id); }}
                  title="删除"
                >
                  ×
                </button>
              </div>
            ))}
          </div>
        )}
      </PageLayout>
    );
  }

  return (
    <PageLayout title="我的学习资料" actions={actions}>
      {recentFolders.length > 0 && (
        <div className="folder-section">
          <h3 className="folder-section-title">最近访问</h3>
          <div className="folder-grid">
            {recentFolders.map((f) => (
              <div
                key={f}
                className="folder-card folder-card-recent"
                onClick={() => handleFolderClick(f)}
              >
                <div className="folder-card-icon">📂</div>
                <div className="folder-card-name">{formatFolderName(f)}</div>
              </div>
            ))}
          </div>
        </div>
      )}

      <div className="folder-section">
        <h3 className="folder-section-title">
          所有文件夹
          <span className="folder-count">{folders.length}</span>
        </h3>
        {folders.length === 0 && !loading ? (
          <div className="material-list-empty">
            暂无文件夹，请上传视频或同步 MinIO
          </div>
        ) : (
          <div className="folder-grid">
            {folders.map((f) => (
              <div
                key={f.name}
                className="folder-card"
                onClick={() => handleFolderClick(f.name)}
              >
                <div className="folder-card-icon">📁</div>
                <div className="folder-card-name">{formatFolderName(f.name)}</div>
                <div className="folder-card-count">{f.count} 个视频</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </PageLayout>
  );
}

export default MaterialList;
