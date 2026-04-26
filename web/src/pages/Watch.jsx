import { useEffect, useRef, useState, useCallback } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import videojs from 'video.js';
import 'video.js/dist/video-js.css';
import { useToast } from '../context/ToastContext';
import { getMaterial, getSubtitle, getVideoStreamUrl, getMaterials } from '../api/material';
import { Button } from '../components/common';
import './Watch.css';

const getProgressKey = (id) => `video_progress_${id}`;
const PLAYBACK_RATE_KEY = 'video_playback_rate';
const AUTO_PLAY_KEY = 'auto_play_next';

function Watch() {
  const { id } = useParams();
  const navigate = useNavigate();
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
  const [transcoding, setTranscoding] = useState(false);
  const transcodedRef = useRef(false);
  const toast = useToast();

  const [folderVideos, setFolderVideos] = useState([]);
  const [showPlaylist, setShowPlaylist] = useState(false);
  const [autoPlay, setAutoPlay] = useState(() => localStorage.getItem(AUTO_PLAY_KEY) !== 'false');

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

        if (materialRes.data?.folder) {
          try {
            const folderRes = await getMaterials(materialRes.data.folder);
            setFolderVideos(folderRes.data || []);
          } catch {
            setFolderVideos([]);
          }
        } else {
          setFolderVideos([]);
        }
      } catch (err) {
        setError('加载失败');
        setShowRetry(true);
      } finally {
        setLoading(false);
      }
    };

    fetchData();
  }, [id]);

  const currentVideoIndex = folderVideos.findIndex(v => String(v.id) === String(id));
  const nextVideo = currentVideoIndex >= 0 && currentVideoIndex < folderVideos.length - 1
    ? folderVideos[currentVideoIndex + 1]
    : null;
  const prevVideo = currentVideoIndex > 0
    ? folderVideos[currentVideoIndex - 1]
    : null;

  const handlePlayNext = useCallback(() => {
    if (nextVideo) {
      navigate(`/watch/${nextVideo.id}`);
    }
  }, [nextVideo, navigate]);

  const handlePlayPrev = useCallback(() => {
    if (prevVideo) {
      navigate(`/watch/${prevVideo.id}`);
    }
  }, [prevVideo, navigate]);

  const handleVideoSwitch = useCallback((videoId) => {
    navigate(`/watch/${videoId}`);
  }, [navigate]);

  const toggleAutoPlay = () => {
    const newVal = !autoPlay;
    setAutoPlay(newVal);
    localStorage.setItem(AUTO_PLAY_KEY, String(newVal));
  };

  useEffect(() => {
    if (!material?.video_url || !videoRef.current) return;

    const getVideoType = (url) => {
      const ext = url.split('.').pop()?.toLowerCase();
      const types = {
        mp4: 'video/mp4',
        webm: 'video/webm',
        ogg: 'video/ogg',
        ogv: 'video/ogg',
      };
      return types[ext] || 'video/mp4';
    };

    const videoType = getVideoType(material.video_url);
    const streamUrl = getVideoStreamUrl(id);

    if (playerRef.current && !playerRef.current.isDisposed()) {
      const player = playerRef.current;
      const currentSrc = player.currentSrc();

      if (currentSrc !== streamUrl) {
        const currentTime = player.currentTime();
        player.src({ src: streamUrl, type: videoType });
        if (currentTime > 0) {
          player.one('loadedmetadata', () => {
            player.currentTime(currentTime);
          });
        }
      }
      return;
    }

    const savedRate = parseFloat(localStorage.getItem(PLAYBACK_RATE_KEY)) || 1;

    const player = videojs(videoRef.current, {
      controls: true,
      fluid: true,
      responsive: true,
      preload: 'auto',
      playbackRates: [0.5, 0.75, 1, 1.25, 1.5, 2],
    });

    player.playbackRate(savedRate);
    player.src({ type: videoType, src: streamUrl });

    player.on('error', () => {
      const err = player.error();
      if (err) {
        if (err.code === 4 && !transcodedRef.current) {
          transcodedRef.current = true;
          setTranscoding(true);
          const transcodedUrl = streamUrl + (streamUrl.includes('?') ? '&' : '?') + 'transcode=1';
          player.src({ type: 'video/mp4', src: transcodedUrl });
          return;
        }
        setError(`视频加载失败: ${err.message || '未知错误'}`);
        setShowRetry(true);
      }
    });

    player.one('loadedmetadata', () => {
      setPlayerReady(true);
      setTranscoding(false);

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

    player.on('ratechange', () => {
      localStorage.setItem(PLAYBACK_RATE_KEY, player.playbackRate().toString());
    });

    player.on('ended', () => {
      setIsEnded(true);
      localStorage.removeItem(getProgressKey(id));
      if (autoPlay && nextVideo) {
        setTimeout(() => handlePlayNext(), 1500);
      }
    });

    player.on('play', () => {
      setIsEnded(false);
    });

    playerRef.current = player;

    player.on('timeupdate', () => {
      const currentTime = player.currentTime();
      const currentSubs = subtitlesRef.current;

      const index = currentSubs.findIndex(
        (sub) => currentTime >= sub.start_time && currentTime <= sub.end_time
      );

      setCurrentSubtitleIndex((prevIndex) => {
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
        const currentTime = playerRef.current.currentTime();
        if (currentTime > 0) {
          localStorage.setItem(getProgressKey(id), currentTime.toString());
        }
        playerRef.current.dispose();
        playerRef.current = null;
        setPlayerReady(false);
      }
      progressRestoredRef.current = false;
      transcodedRef.current = false;
    };
  }, [material, id, autoPlay, nextVideo, handlePlayNext]);

  useEffect(() => {
    if (currentSubtitleIndex >= 0 && subtitleListRef.current && subtitleExpanded && !searchQuery) {
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
          container.scrollTo({
            top: elementTop - containerHeight / 2 + elementHeight / 2,
            behavior: 'smooth',
          });
        }
      }
    }
  }, [currentSubtitleIndex, subtitleExpanded, searchQuery]);

  useEffect(() => {
    const handleKeyDown = (e) => {
      const player = playerRef.current;
      if (!player || !playerReady || player.isDisposed()) return;

      if (e.target.tagName === 'INPUT' || e.target.tagName === 'TEXTAREA') {
        if (e.key !== 'Escape') return;
      }

      switch (e.key) {
        case ' ':
          e.preventDefault();
          player.paused() ? player.play() : player.pause();
          break;
        case 'ArrowLeft':
          e.preventDefault();
          player.currentTime(Math.max(0, player.currentTime() - 10));
          break;
        case 'ArrowRight':
          e.preventDefault();
          player.currentTime(player.currentTime() + 10);
          break;
        case 'ArrowUp':
          e.preventDefault();
          player.volume(Math.min(1, player.volume() + 0.1));
          break;
        case 'ArrowDown':
          e.preventDefault();
          player.volume(Math.max(0, player.volume() - 0.1));
          break;
        case 'f':
        case 'F':
          e.preventDefault();
          player.isFullscreen() ? player.exitFullscreen() : player.requestFullscreen();
          break;
        case 'm':
        case 'M':
          e.preventDefault();
          player.muted(!player.muted());
          break;
        case '/':
        case '?':
          e.preventDefault();
          setShowSearch(true);
          setTimeout(() => searchInputRef.current?.focus(), 100);
          break;
        case 's':
        case 'S':
          e.preventDefault();
          setSubtitleExpanded(!subtitleExpanded);
          break;
        case 'c':
        case 'C':
          e.preventDefault();
          setShowOverlaySubtitle(!showOverlaySubtitle);
          break;
        case 'p':
        case 'P':
          e.preventDefault();
          setShowPlaylist(!showPlaylist);
          break;
        case 'n':
        case 'N':
          e.preventDefault();
          if (nextVideo) handlePlayNext();
          break;
        case 'Escape':
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
  }, [playerReady, subtitleExpanded, showSearch, showOverlaySubtitle, subtitles, showPlaylist, nextVideo, handlePlayNext]);

  const handleSubtitleClick = useCallback((startTime) => {
    const player = playerRef.current;
    if (!player || player.isDisposed() || !playerReady) return;

    try {
      const duration = player.duration();
      if (duration > 0 && startTime > duration) return;

      player.pause();
      player.currentTime(startTime);
      player.play();
      setIsEnded(false);
    } catch (e) {
      console.error('Failed to seek video:', e);
    }
  }, [playerReady]);

  const handleSearch = (query) => {
    setSearchQuery(query);
    if (!query.trim()) {
      setFilteredSubtitles(subtitles);
    } else {
      setFilteredSubtitles(
        subtitles.filter((sub) => sub.text.toLowerCase().includes(query.toLowerCase()))
      );
    }
  };

  const handleRetry = () => {
    setError('');
    setShowRetry(false);
    setLoading(true);
    window.location.reload();
  };

  const handleReplay = () => {
    const player = playerRef.current;
    if (player && !player.isDisposed()) {
      player.currentTime(0);
      player.play();
      setIsEnded(false);
    }
  };

  if (loading) {
    return (
      <div className="watch-loading">
        <div className="watch-spinner" />
        <p>加载中...</p>
      </div>
    );
  }

  if (error || !material) {
    return (
      <div className="watch-error">
        <p>{error || '资料不存在'}</p>
        <div className="watch-error-actions">
          {showRetry && (
            <Button variant="primary" onClick={handleRetry}>重试</Button>
          )}
          <Link to="/">
            <Button variant="secondary">返回列表</Button>
          </Link>
        </div>
      </div>
    );
  }

  if (!material.video_url) {
    return (
      <div className="watch-error">
        <p>视频资源不可用</p>
        <Link to="/">
          <Button variant="primary">返回列表</Button>
        </Link>
      </div>
    );
  }

  const displaySubtitles = searchQuery ? filteredSubtitles : subtitles;

  return (
    <div className="watch-page">
      <header className="watch-header">
        <Link to="/" className="watch-back">← 返回</Link>
        <h1 className="watch-title">{material.title}</h1>
        <div className="watch-actions">
          {folderVideos.length > 0 && (
            <button
              className={`watch-action-btn ${showPlaylist ? 'active' : ''}`}
              onClick={() => setShowPlaylist(!showPlaylist)}
              title="播放列表 (P)"
            >
              📋
            </button>
          )}
          <button
            className="watch-action-btn"
            onClick={() => setShowOverlaySubtitle(!showOverlaySubtitle)}
            title={`${showOverlaySubtitle ? '隐藏' : '显示'}画面字幕 (C)`}
          >
            {showOverlaySubtitle ? '💬' : '🚫'}
          </button>
          <button
            className="watch-action-btn"
            onClick={() => setShowSearch(!showSearch)}
            title="搜索字幕 (/)"
          >
            🔍
          </button>
          <button
            className="watch-action-btn"
            onClick={() => setSubtitleExpanded(!subtitleExpanded)}
            title={`${subtitleExpanded ? '收起' : '展开'}字幕面板 (S)`}
          >
            {subtitleExpanded ? '📑' : '📄'}
          </button>
        </div>
      </header>

      <div className="watch-container">
        <div className="watch-video-section">
          <div className="watch-video-wrapper">
            <video ref={videoRef} className="video-js vjs-big-play-centered" />

            {transcoding && (
              <div className="watch-transcoding-overlay">
                <div className="watch-spinner" />
                <p>正在转码，请稍候...</p>
              </div>
            )}

            {showOverlaySubtitle && currentSubtitleText && (
              <div className="watch-overlay-subtitle">
                {currentSubtitleText.split('\n').map((line, i) => (
                  <p key={i}>{line}</p>
                ))}
              </div>
            )}

            {isEnded && (
              <div className="watch-ended-overlay">
                <div className="watch-ended-content">
                  <div className="watch-ended-icon">▶</div>
                  <p>视频播放完毕</p>
                  <div className="watch-ended-actions">
                    <Button variant="primary" onClick={handleReplay}>↺ 重播</Button>
                    {nextVideo && (
                      <Button variant="primary" onClick={handlePlayNext}>
                        下一集 ▶
                      </Button>
                    )}
                    <Link to="/">
                      <Button variant="secondary">返回列表</Button>
                    </Link>
                  </div>
                </div>
              </div>
            )}
          </div>

          <div className="watch-nav-bar">
            <button
              className="watch-nav-btn"
              onClick={handlePlayPrev}
              disabled={!prevVideo}
              title="上一集"
            >
              ◀ 上一集
            </button>
            <label className="watch-autoplay-toggle" title="自动连播">
              <input
                type="checkbox"
                checked={autoPlay}
                onChange={toggleAutoPlay}
              />
              <span>自动连播</span>
            </label>
            <button
              className="watch-nav-btn"
              onClick={handlePlayNext}
              disabled={!nextVideo}
              title="下一集 (N)"
            >
              下一集 ▶
            </button>
          </div>

          {material.description && (
            <div className="watch-description">
              <p>{material.description}</p>
            </div>
          )}

          <div className="watch-shortcuts">
            <span>空格 暂停</span>
            <span>← → 快进/退</span>
            <span>F 全屏</span>
            <span>P 播放列表</span>
            <span>N 下一集</span>
          </div>
        </div>

        <div className="watch-sidebar">
          {showPlaylist && folderVideos.length > 0 && (
            <div className="watch-playlist-panel">
              <div className="playlist-header">
                <h3>播放列表</h3>
                <span className="playlist-count">{currentVideoIndex + 1} / {folderVideos.length}</span>
              </div>
              <div className="playlist-list">
                {folderVideos.map((v, idx) => (
                  <div
                    key={v.id}
                    className={`playlist-item ${String(v.id) === String(id) ? 'active' : ''}`}
                    onClick={() => handleVideoSwitch(v.id)}
                  >
                    <span className="playlist-item-index">{idx + 1}</span>
                    <span className="playlist-item-title">{v.title}</span>
                    {v.has_subtitle && <span className="playlist-item-badge">字幕</span>}
                  </div>
                ))}
              </div>
            </div>
          )}

          {subtitles.length > 0 && (
            <div className={`watch-subtitle-panel ${subtitleExpanded ? 'expanded' : 'collapsed'}`}>
              <div
                className="watch-subtitle-header"
                onClick={() => setSubtitleExpanded(!subtitleExpanded)}
              >
                <h3>字幕</h3>
                <button className="watch-subtitle-toggle">
                  {subtitleExpanded ? '−' : '+'}
                </button>
              </div>

              {subtitleExpanded && (
                <>
                  {showSearch && (
                    <div className="watch-subtitle-search">
                      <input
                        ref={searchInputRef}
                        type="text"
                        placeholder="搜索字幕..."
                        value={searchQuery}
                        onChange={(e) => handleSearch(e.target.value)}
                        onKeyDown={(e) => {
                          if (e.key === 'Escape') {
                            setShowSearch(false);
                            setSearchQuery('');
                            setFilteredSubtitles(subtitles);
                          }
                        }}
                      />
                      {searchQuery && (
                        <button
                          className="watch-search-clear"
                          onClick={() => {
                            setSearchQuery('');
                            setFilteredSubtitles(subtitles);
                          }}
                        >
                          ✕
                        </button>
                      )}
                    </div>
                  )}

                  <div className="watch-subtitle-list" ref={subtitleListRef}>
                    {displaySubtitles.length === 0 ? (
                      <div className="watch-no-results">无匹配字幕</div>
                    ) : (
                      displaySubtitles.map((subtitle, index) => {
                        const originalIndex = subtitles.findIndex((s) => s.index === subtitle.index);
                        return (
                          <div
                            key={subtitle.index || index}
                            className={`watch-subtitle-item ${
                              originalIndex === currentSubtitleIndex ? 'active' : ''
                            }`}
                            onClick={() => handleSubtitleClick(subtitle.start_time)}
                          >
                            <span className="watch-subtitle-time">
                              {formatTime(subtitle.start_time)}
                            </span>
                            <p className="watch-subtitle-text">{subtitle.text}</p>
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
    </div>
  );
}

function formatTime(seconds) {
  const mins = Math.floor(seconds / 60);
  const secs = Math.floor(seconds % 60);
  return `${mins.toString().padStart(2, '0')}:${secs.toString().padStart(2, '0')}`;
}

export default Watch;
