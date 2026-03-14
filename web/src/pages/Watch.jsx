import { useEffect, useRef, useState, useCallback } from 'react';
import { useParams, Link } from 'react-router-dom';
import videojs from 'video.js';
import 'video.js/dist/video-js.css';
import { getMaterial, getSubtitle, getVideoStreamUrl } from '../api/material';

// 进度存储 key
const getProgressKey = (id) => `video_progress_${id}`;
const PLAYBACK_RATE_KEY = 'video_playback_rate';

function Watch() {
  const { id } = useParams();
  const videoRef = useRef(null);
  const playerRef = useRef(null);
  const subtitleListRef = useRef(null);
  const subtitlesRef = useRef([]);
  const progressSaveTimerRef = useRef(null);
  const searchInputRef = useRef(null);
  const progressRestoredRef = useRef(false);
  
  const [material, setMaterial] = useState(null);
  const [subtitles, setSubtitles] = useState([]);
  const [filteredSubtitles, setFilteredSubtitles] = useState([]);
  const [currentSubtitleIndex, setCurrentSubtitleIndex] = useState(-1);
  const [currentSubtitleText, setCurrentSubtitleText] = useState('');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');
  const [subtitleExpanded, setSubtitleExpanded] = useState(true);
  const [playerReady, setPlayerReady] = useState(false);
  const [showRetry, setShowRetry] = useState(false);
  const [isEnded, setIsEnded] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [showSearch, setShowSearch] = useState(false);
  const [showOverlaySubtitle, setShowOverlaySubtitle] = useState(true);

  // 加载资料和字幕
  useEffect(() => {
    const fetchData = async () => {
      try {
        const [materialRes, subtitleRes] = await Promise.all([
          getMaterial(id),
          getSubtitle(id).catch(() => ({ data: [] })),
        ]);

        setMaterial(materialRes.data);
        setSubtitles(subtitleRes.data);
        setFilteredSubtitles(subtitleRes.data);
        subtitlesRef.current = subtitleRes.data;
      } catch (err) {
        setError('加载失败');
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [id]);

  // 初始化 Video.js 播放器
  useEffect(() => {
    if (!material?.video_url || !videoRef.current) return;

    const getVideoType = (url) => {
      const ext = url.split('.').pop()?.toLowerCase();
      const types = {
        'mp4': 'video/mp4',
        'webm': 'video/webm',
        'ogg': 'video/ogg',
        'ogv': 'video/ogg',
        'mov': 'video/quicktime',
        'mkv': 'video/x-matroska',
        'avi': 'video/x-msvideo',
      };
      return types[ext] || 'video/mp4';
    };

    const videoType = getVideoType(material.video_url);
    const isSupported = ['video/mp4', 'video/webm', 'video/ogg'].includes(videoType);

    if (!isSupported) {
      setError(`视频格式 .${videoType.split('/').pop()} 不被浏览器支持，请转换为 MP4 格式`);
      return;
    }

    const streamUrl = getVideoStreamUrl(id);

    // 如果播放器已存在，只更新视频源
    if (playerRef.current && !playerRef.current.isDisposed()) {
      const player = playerRef.current;
      const currentSrc = player.currentSrc();
      
      // 只有当视频源真正改变时才更新
      if (currentSrc !== streamUrl) {
        // 记录当前播放进度
        const currentTime = player.currentTime();
        
        player.src({
          src: streamUrl,
          type: videoType,
        });
        
        // 恢复播放进度（如果之前有进度）
        if (currentTime > 0) {
          player.one('loadedmetadata', () => {
            player.currentTime(currentTime);
          });
        }
      }
      return;
    }

    // 获取保存的播放速度
    const savedRate = parseFloat(localStorage.getItem(PLAYBACK_RATE_KEY)) || 1;

    const player = videojs(videoRef.current, {
      controls: true,
      fluid: true,
      responsive: true,
      preload: 'auto',
      playbackRates: [0.5, 0.75, 1, 1.25, 1.5, 2],
    });

    // 恢复播放速度
    player.playbackRate(savedRate);

    // 设置视频源
    console.log('Setting video source:', streamUrl, 'type:', videoType);
    player.src({
      type: videoType,
      src: streamUrl,
    });

    // 监听播放错误
    player.on('error', () => {
      const err = player.error();
      console.error('Video error:', err);
      if (err) {
        const errorMessages = {
          1: '视频加载被中断',
          2: '网络错误，无法加载视频',
          3: '视频解码错误',
          4: '视频格式不支持或文件不存在',
          5: '视频加密或受保护',
        };
        setError(errorMessages[err.code] || `视频加载失败: ${err.message || '未知错误'}`);
        setShowRetry(true);
      }
    });

    // 监听加载事件
    player.on('loadstart', () => console.log('Video: loadstart'));
    player.on('loadeddata', () => console.log('Video: loadeddata'));
    player.on('canplay', () => console.log('Video: canplay'));
    player.on('waiting', () => console.log('Video: waiting/buffering'));
    player.on('playing', () => console.log('Video: playing'));

    // 播放器加载元数据后标记为就绪
    player.one('loadedmetadata', () => {
      setPlayerReady(true);
      
      // 恢复播放进度（只恢复一次，避免覆盖用户手动设置的进度）
      if (!progressRestoredRef.current) {
        progressRestoredRef.current = true;
        const savedProgress = localStorage.getItem(getProgressKey(id));
        if (savedProgress) {
          const time = parseFloat(savedProgress);
          const duration = player.duration();
          if (time > 0 && duration > 0 && time < duration - 10) {
            player.currentTime(time);
          }
        }
      }
    });

    // 监听播放速度变化并保存
    player.on('ratechange', () => {
      const rate = player.playbackRate();
      localStorage.setItem(PLAYBACK_RATE_KEY, rate.toString());
    });

    // 监听播放结束
    player.on('ended', () => {
      setIsEnded(true);
      localStorage.removeItem(getProgressKey(id));
    });

    // 监听播放开始（隐藏结束提示）
    player.on('play', () => {
      setIsEnded(false);
    });

    playerRef.current = player;

    // 监听时间更新
    player.on('timeupdate', () => {
      const currentTime = player.currentTime();
      const currentSubs = subtitlesRef.current;
      
      // 同步字幕高亮
      const index = currentSubs.findIndex(
        (sub) => currentTime >= sub.start_time && currentTime <= sub.end_time
      );

      setCurrentSubtitleIndex(prevIndex => {
        if (index !== prevIndex) {
          if (index >= 0) {
            setCurrentSubtitleText(currentSubs[index]?.text || '');
          } else {
            setCurrentSubtitleText('');
          }
          return index;
        }
        return prevIndex;
      });

      // 定期保存进度（每5秒）
      if (!progressSaveTimerRef.current) {
        progressSaveTimerRef.current = setTimeout(() => {
          if (player && !player.isDisposed() && !player.paused()) {
            localStorage.setItem(getProgressKey(id), currentTime.toString());
          }
          progressSaveTimerRef.current = null;
        }, 5000);
      }
    });

    return () => {
      if (progressSaveTimerRef.current) {
        clearTimeout(progressSaveTimerRef.current);
      }
      if (playerRef.current && !playerRef.current.isDisposed()) {
        // 保存最终进度
        const currentTime = playerRef.current.currentTime();
        if (currentTime > 0) {
          localStorage.setItem(getProgressKey(id), currentTime.toString());
        }
        playerRef.current.dispose();
        playerRef.current = null;
        setPlayerReady(false);
      }
      // 重置进度恢复标记，以便下次可以恢复新视频的进度
      progressRestoredRef.current = false;
    };
  }, [material, id]);

  // 字幕滚动到当前位置
  useEffect(() => {
    if (
      currentSubtitleIndex >= 0 &&
      subtitleListRef.current &&
      subtitleExpanded &&
      !searchQuery
    ) {
      const activeElement = subtitleListRef.current.children[currentSubtitleIndex];
      if (activeElement) {
        const container = subtitleListRef.current;
        const elementTop = activeElement.offsetTop;
        const elementHeight = activeElement.offsetHeight;
        const containerHeight = container.clientHeight;
        const scrollTop = container.scrollTop;

        const elementVisibleTop = elementTop - scrollTop;
        const elementVisibleBottom = elementVisibleTop + elementHeight;

        if (elementVisibleTop < 50 || elementVisibleBottom > containerHeight - 50) {
          const targetScrollTop = elementTop - containerHeight / 2 + elementHeight / 2;
          container.scrollTo({
            top: targetScrollTop,
            behavior: 'smooth',
          });
        }
      }
    }
  }, [currentSubtitleIndex, subtitleExpanded, searchQuery]);

  // 键盘快捷键
  useEffect(() => {
    const handleKeyDown = (e) => {
      const player = playerRef.current;
      if (!player || !playerReady || player.isDisposed()) return;

      // 如果正在输入框中输入，不处理快捷键
      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
        if (e.key !== 'Escape') return;
      }

      switch (e.key) {
        case ' ': // 空格：暂停/播放
          e.preventDefault();
          if (player.paused()) {
            player.play();
          } else {
            player.pause();
          }
          break;
        case 'ArrowLeft': // 左箭头：后退10秒
          e.preventDefault();
          player.currentTime(Math.max(0, player.currentTime() - 10));
          break;
        case 'ArrowRight': // 右箭头：前进10秒
          e.preventDefault();
          player.currentTime(player.currentTime() + 10);
          break;
        case 'ArrowUp': // 上箭头：增加音量
          e.preventDefault();
          player.volume(Math.min(1, player.volume() + 0.1));
          break;
        case 'ArrowDown': // 下箭头：减小音量
          e.preventDefault();
          player.volume(Math.max(0, player.volume() - 0.1));
          break;
        case 'f': // F：全屏切换
        case 'F':
          e.preventDefault();
          if (player.isFullscreen()) {
            player.exitFullscreen();
          } else {
            player.requestFullscreen();
          }
          break;
        case 'm': // M：静音切换
        case 'M':
          e.preventDefault();
          player.muted(!player.muted());
          break;
        case '/': // /：搜索字幕
        case '?':
          e.preventDefault();
          setShowSearch(true);
          setTimeout(() => searchInputRef.current?.focus(), 100);
          break;
        case 's': // S：切换字幕面板
        case 'S':
          e.preventDefault();
          setSubtitleExpanded(!subtitleExpanded);
          break;
        case 'c': // C：切换画面字幕
        case 'C':
          e.preventDefault();
          setShowOverlaySubtitle(!showOverlaySubtitle);
          break;
        case 'Escape': // ESC：关闭搜索
          if (showSearch) {
            setShowSearch(false);
            setSearchQuery('');
            setFilteredSubtitles(subtitles);
          }
          break;
        default:
          break;
      }
    };

    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, [playerReady, subtitleExpanded, showSearch, showOverlaySubtitle, subtitles]);

  // 点击字幕跳转到对应时间
  const handleSubtitleClick = useCallback((startTime) => {
    const player = playerRef.current;
    if (!player) {
      console.warn('Player not initialized');
      return;
    }
    if (player.isDisposed()) {
      console.warn('Player is disposed');
      return;
    }
    if (!playerReady) {
      console.warn('Player not ready yet');
      return;
    }
    
    try {
      const duration = player.duration();
      if (duration > 0 && startTime > duration) {
        console.warn('Seek time exceeds video duration');
        return;
      }
      
      // 先暂停再跳转再播放，确保跳转成功
      player.pause();
      player.currentTime(startTime);
      player.play();
      setIsEnded(false);
    } catch (e) {
      console.error('Failed to seek video:', e);
    }
  }, [playerReady]);

  // 搜索字幕
  const handleSearch = (query) => {
    setSearchQuery(query);
    if (!query.trim()) {
      setFilteredSubtitles(subtitles);
    } else {
      const filtered = subtitles.filter(sub => 
        sub.text.toLowerCase().includes(query.toLowerCase())
      );
      setFilteredSubtitles(filtered);
    }
  };

  // 重试加载
  const handleRetry = () => {
    setError('');
    setShowRetry(false);
    setLoading(true);
    window.location.reload();
  };

  // 重播
  const handleReplay = () => {
    const player = playerRef.current;
    if (player && !player.isDisposed()) {
      player.currentTime(0);
      player.play();
      setIsEnded(false);
    }
  };

  if (loading) {
    return <div className="loading">加载中...</div>;
  }

  if (error || !material) {
    return (
      <div className="error-container">
        <p>{error || '资料不存在'}</p>
        {showRetry && (
          <button onClick={handleRetry} className="btn-primary" style={{ marginTop: '16px' }}>
            重试
          </button>
        )}
        <Link to="/" className="btn-primary" style={{ marginTop: showRetry ? '12px' : '16px' }}>
          返回列表
        </Link>
      </div>
    );
  }

  if (!material.video_url) {
    return (
      <div className="error-container">
        <p>视频资源不可用</p>
        <Link to="/" className="btn-primary">
          返回列表
        </Link>
      </div>
    );
  }

  const displaySubtitles = searchQuery ? filteredSubtitles : subtitles;

  return (
    <div className="watch-page">
      <header className="watch-header">
        <Link to="/" className="btn-back">
          ← 返回
        </Link>
        <h1>{material.title}</h1>
        <div className="header-actions">
          <button 
            className="btn-icon" 
            onClick={() => setShowOverlaySubtitle(!showOverlaySubtitle)}
            title={`${showOverlaySubtitle ? '隐藏' : '显示'}画面字幕 (C)`}
          >
            {showOverlaySubtitle ? '💬' : '🚫'}
          </button>
          <button 
            className="btn-icon" 
            onClick={() => setShowSearch(!showSearch)}
            title="搜索字幕 (/)"
          >
            🔍
          </button>
          <button 
            className="btn-icon" 
            onClick={() => setSubtitleExpanded(!subtitleExpanded)}
            title={`${subtitleExpanded ? '收起' : '展开'}字幕面板 (S)`}
          >
            {subtitleExpanded ? '📑' : '📄'}
          </button>
        </div>
      </header>

      <div className="watch-container">
        {/* 视频区域 */}
        <div className="video-section">
          <div className="video-wrapper">
            <video
              ref={videoRef}
              className="video-js vjs-big-play-centered"
            />
            
            {/* 画面上字幕叠加 */}
            {showOverlaySubtitle && currentSubtitleText && (
              <div className="overlay-subtitle">
                {currentSubtitleText.split('\n').map((line, i) => (
                  <p key={i}>{line}</p>
                ))}
              </div>
            )}

            {/* 播放结束提示 */}
            {isEnded && (
              <div className="video-ended-overlay">
                <div className="ended-content">
                  <div className="ended-icon">▶</div>
                  <p>视频播放完毕</p>
                  <div className="ended-actions">
                    <button onClick={handleReplay} className="btn-primary">
                      ↺ 重播
                    </button>
                    <Link to="/" className="btn-secondary">
                      返回列表
                    </Link>
                  </div>
                </div>
              </div>
            )}
          </div>

          {material.description && (
            <div className="video-description">
              <p>{material.description}</p>
            </div>
          )}

          {/* 快捷键提示 */}
          <div className="shortcuts-hint">
            <span>空格 暂停 | ← → 快进/后退 | ↑ ↓ 音量 | F 全屏 | / 搜索</span>
          </div>
        </div>

        {/* 字幕侧边栏 */}
        {subtitles.length > 0 && (
          <div className={`subtitle-panel ${subtitleExpanded ? 'expanded' : 'collapsed'}`}>
            <div className="subtitle-header" onClick={() => setSubtitleExpanded(!subtitleExpanded)}>
              <h3>字幕</h3>
              <button className="subtitle-toggle">
                {subtitleExpanded ? '−' : '+'}
              </button>
            </div>
            
            {subtitleExpanded && (
              <>
                {/* 搜索框 */}
                {showSearch && (
                  <div className="subtitle-search">
                    <input
                      ref={searchInputRef}
                      type="text"
                      placeholder="搜索字幕..."
                      value={searchQuery}
                      onChange={(e) => handleSearch(e.target.value)}
                      onKeyDown={(e) => e.key === 'Escape' && (setShowSearch(false), setSearchQuery(''), setFilteredSubtitles(subtitles))}
                    />
                    {searchQuery && (
                      <button
                        className="search-clear"
                        onClick={() => { setSearchQuery(''); setFilteredSubtitles(subtitles); }}
                        title="清空搜索"
                      >
                        ✕
                      </button>
                    )}
                    {!searchQuery && (
                      <span className="search-count">
                        {filteredSubtitles.length} 条
                      </span>
                    )}
                  </div>
                )}
                
                <div className="subtitle-list" ref={subtitleListRef}>
                  {displaySubtitles.length === 0 ? (
                    <div className="no-results">无匹配字幕</div>
                  ) : (
                    displaySubtitles.map((subtitle, index) => {
                      // 找到原数组中的索引用于高亮
                      const originalIndex = subtitles.findIndex(s => s.index === subtitle.index);
                      return (
                        <div
                          key={subtitle.index || index}
                          className={`subtitle-item ${
                            originalIndex === currentSubtitleIndex ? 'active' : ''
                          }`}
                          onClick={() => handleSubtitleClick(subtitle.start_time)}
                        >
                          <span className="subtitle-time">
                            {formatTime(subtitle.start_time)}
                          </span>
                          <p className="subtitle-text">{subtitle.text}</p>
                        </div>
                      );
                    })
                  )}
                </div>
              </>
            )}
          </div>
        )}
      </div>
    </div>
  );
}

// 格式化时间为 MM:SS
function formatTime(seconds) {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins.toString().padStart(2, '0')}:${secs
    .toString()
    .padStart(2, '0')}`;
}

export default Watch;