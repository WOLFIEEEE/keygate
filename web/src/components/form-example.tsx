import { ArrowRight, Check, Mail, RotateCcw } from "lucide-react"
import { type FormEvent, useRef, useState } from "react"

type Errors = Partial<Record<"name" | "email" | "message", string>>

export function FormExample() {
  const [conditional, setConditional] = useState(false)
  const [project, setProject] = useState(false)
  const [errors, setErrors] = useState<Errors>({})
  const [submitted, setSubmitted] = useState(false)
  const summary = useRef<HTMLDivElement>(null)
  const form = useRef<HTMLFormElement>(null)
  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault()
    const data = new FormData(event.currentTarget)
    const next: Errors = {}
    if (!String(data.get("name") || "").trim()) next.name = "Enter a name for this example."
    const email = event.currentTarget.elements.namedItem("email") as HTMLInputElement
    if (!email.value.trim()) next.email = "Enter an example email address."
    else if (email.validity.typeMismatch) next.email = "Enter an email address in the format name@example.com."
    if (!String(data.get("message") || "").trim()) next.message = "Add a message for this example."
    setErrors(next)
    if (Object.keys(next).length) requestAnimationFrame(() => summary.current?.focus())
    else {
      setSubmitted(true)
      requestAnimationFrame(() => document.getElementById("example-success")?.focus())
    }
  }
  function reset() {
    form.current?.reset()
    setProject(false)
    setErrors({})
    setSubmitted(false)
  }
  return (
    <div className="af-example">
      <div className="af-example-top">
        <span>
          <span className="af-example-dot" />
          Interactive example
        </span>
        <span>WordPress forms</span>
      </div>
      <fieldset className="af-example-toolbar">
        <legend className="af-sr-only">Choose a form example</legend>
        <button
          type="button"
          aria-pressed={!conditional}
          onClick={() => {
            reset()
            setConditional(false)
          }}
        >
          Contact form <span>Free</span>
        </button>
        <button
          type="button"
          aria-pressed={conditional}
          onClick={() => {
            reset()
            setConditional(true)
          }}
        >
          Conditional questions <span>Pro</span>
        </button>
      </fieldset>
      {submitted ? (
        <div className="af-example-success">
          <Check aria-hidden="true" size={32} />
          <h2 id="example-success" tabIndex={-1}>
            Your example is complete.
          </h2>
          <p role="status">The answers passed validation. This preview has not sent or saved them.</p>
          <button type="button" className="af-button af-button-secondary" onClick={reset}>
            Try another example <RotateCcw aria-hidden="true" size={16} />
          </button>
        </div>
      ) : (
        <form
          ref={form}
          onSubmit={submit}
          noValidate
          className="af-example-form"
          key={conditional ? "conditional" : "contact"}
        >
          <h2>Let’s talk about your project.</h2>
          <p className="af-example-intro">Try the labels, keyboard focus and linked errors.</p>
          {Object.keys(errors).length > 0 && (
            <div className="af-error-summary" role="alert" tabIndex={-1} ref={summary}>
              <p>
                <strong>Check these answers</strong>
              </p>
              <ul>
                {Object.entries(errors).map(([key, text]) => (
                  <li key={key}>
                    <a href={`#example-${key}`}>{text}</a>
                  </li>
                ))}
              </ul>
            </div>
          )}
          <div className="af-example-fields">
            <div className="af-field">
              <label htmlFor="example-name">
                Your name <span>(required)</span>
              </label>
              <input
                id="example-name"
                name="name"
                autoComplete="off"
                required
                aria-invalid={!!errors.name}
                aria-describedby={errors.name ? "example-name-error" : undefined}
              />
              {errors.name && (
                <p id="example-name-error" className="af-field-error">
                  {errors.name}
                </p>
              )}
            </div>
            <div className="af-field">
              <label htmlFor="example-email">
                Email address <span>(required)</span>
              </label>
              <input
                id="example-email"
                name="email"
                type="email"
                placeholder="you@example.com"
                autoComplete="off"
                required
                aria-invalid={!!errors.email}
                aria-describedby={errors.email ? "example-email-error" : undefined}
              />
              {errors.email && (
                <p id="example-email-error" className="af-field-error">
                  {errors.email}
                </p>
              )}
            </div>
          </div>
          {conditional && (
            <>
              <div className="af-field">
                <label htmlFor="example-request">What would you like to discuss?</label>
                <select
                  id="example-request"
                  name="request"
                  onChange={(event) => setProject(event.target.value === "website")}
                >
                  <option value="general">A general question</option>
                  <option value="website">A website project</option>
                </select>
              </div>
              <p className="af-conditional-status" role="status" aria-live="polite">
                {project
                  ? "A project budget question is now available."
                  : "Choose “A website project” to reveal a relevant question."}
              </p>
              {project && (
                <div className="af-field af-revealed">
                  <label htmlFor="example-budget">Project budget</label>
                  <select id="example-budget" name="budget">
                    <option>Still deciding</option>
                    <option>Under $1,000</option>
                    <option>$1,000–$5,000</option>
                    <option>Over $5,000</option>
                  </select>
                  <p className="af-field-help">This question appears only for website projects.</p>
                </div>
              )}
            </>
          )}
          <div className="af-field">
            <label htmlFor="example-message">
              Your message <span>(required)</span>
            </label>
            <textarea
              id="example-message"
              name="message"
              rows={3}
              required
              aria-invalid={!!errors.message}
              aria-describedby={errors.message ? "example-message-error" : undefined}
            />
            {errors.message && (
              <p id="example-message-error" className="af-field-error">
                {errors.message}
              </p>
            )}
          </div>
          <button className="af-button af-example-submit" type="submit">
            Try this form <ArrowRight aria-hidden="true" size={17} />
          </button>
          <p className="af-example-privacy">
            <Mail aria-hidden="true" size={14} /> Browser-only preview. Nothing is sent or saved.
          </p>
        </form>
      )}
      <div className="af-example-bottom">
        <Check aria-hidden="true" size={15} />
        Visible labels
        <Check aria-hidden="true" size={15} />
        Linked errors
        <Check aria-hidden="true" size={15} />
        Keyboard focus
      </div>
    </div>
  )
}
