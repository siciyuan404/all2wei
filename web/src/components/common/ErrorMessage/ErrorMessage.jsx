import './ErrorMessage.css';

function ErrorMessage({ message, onRetry }) {
  if (!message) return null;

  return (
    <div className="error-container">
      <div className="error-icon">⚠</div>
      <p className="error-text">{message}</p>
      {onRetry && (
        <button className="error-retry" onClick={onRetry}>
          重试
        </button>
      )}
    </div>
  );
}

export default ErrorMessage;
