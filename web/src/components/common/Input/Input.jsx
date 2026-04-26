import './Input.css';

function Input({
  label,
  error,
  type = 'text',
  placeholder,
  value,
  onChange,
  required,
  disabled,
  className = '',
  ...props
}) {
  return (
    <div className={`input-group ${error ? 'input-group-error' : ''} ${className}`}>
      {label && (
        <label className="input-label">
          {label}
          {required && <span className="input-required">*</span>}
        </label>
      )}
      <input
        type={type}
        className="input-field"
        placeholder={placeholder}
        value={value}
        onChange={onChange}
        required={required}
        disabled={disabled}
        {...props}
      />
      {error && <span className="input-error">{error}</span>}
    </div>
  );
}

export default Input;
