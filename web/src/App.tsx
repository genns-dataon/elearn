import { useState, useRef } from 'react'
import {
  Layout,
  Upload,
  Button,
  Card,
  InputNumber,
  Progress,
  Input,
  List,
  Avatar,
  Tag,
  Switch,
  Typography,
  Divider,
  Space,
  message,
  Empty,
  Badge,
  Tooltip,
  Modal,
  Checkbox,
  Select,
} from 'antd'
import {
  UploadOutlined,
  FileTextOutlined,
  SendOutlined,
  LeftOutlined,
  RightOutlined,
  RobotOutlined,
  UserOutlined,
  BulbOutlined,
  MoonOutlined,
  SunOutlined,
  ReadOutlined,
  SoundOutlined,
  DeleteOutlined,
  BookOutlined,
  UnorderedListOutlined,
} from '@ant-design/icons'
import axios from 'axios'

const { Header, Content, Sider } = Layout
const { Title, Text, Paragraph } = Typography
const { TextArea } = Input

const API_BASE = import.meta.env.VITE_API_URL || (window.location.hostname === 'localhost' || window.location.hostname === '127.0.0.1' ? 'http://localhost:8080/api' : '/api')

interface SourceFile {
  id: string
  course_id: string
  filename: string
  file_path: string
  file_size: number
  created_at: string
}

interface Slide {
  id: string
  slide_number: number
  title: string
  content: string
  instructor_script?: string
  image_prompt?: string
  image_url?: string
  audio_url?: string
  layout?: string
  theme?: string
}

interface ChatMessage {
  role: 'user' | 'assistant'
  content: string
  citations?: string[]
}

interface Question {
  slide_id: string
  question: string
  options: string[]
  correct_answer: number
}

interface QuizAnswer {
  slideId: string
  selectedAnswer: number
  isCorrect: boolean
}

function App() {
  const [darkMode, setDarkMode] = useState(false)
  const [courseId, setCourseId] = useState<string | null>(null)
  const [sourceFiles, setSourceFiles] = useState<SourceFile[]>([])
  const [slides, setSlides] = useState<Slide[]>([])
  const [currentSlide, setCurrentSlide] = useState(0)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [question, setQuestion] = useState('')
  const [loading, setLoading] = useState(false)
  const [uploadLoading, setUploadLoading] = useState(false)
  const [genLoading, setGenLoading] = useState(false)

  // Generation settings
  const [genModalVisible, setGenModalVisible] = useState(false)
  const [numSlides, setNumSlides] = useState(10)
  const [presentationStyle, setPresentationStyle] = useState('balanced')
  const [instructorPrompt, setInstructorPrompt] = useState('friendly, conversational, engaging presentation style targeted at a general audience. Do not ask questions at the end of slides.')
  const [generateImages, setGenerateImages] = useState(false)
  const [useWebImages, setUseWebImages] = useState(true)
  const [useDalle, setUseDalle] = useState(false)
  const [generateVoiceover, setGenerateVoiceover] = useState(false)
  const [generateQuestions, setGenerateQuestions] = useState(false)
  const [language, setLanguage] = useState('english')

  const [notesVisible, setNotesVisible] = useState(false)
  const [tocVisible, setTocVisible] = useState(false)
  const [questions, setQuestions] = useState<Question[]>([])
  const [quizAnswers, setQuizAnswers] = useState<QuizAnswer[]>([])
  const chatEndRef = useRef<HTMLDivElement>(null)
  const audioRef = useRef<HTMLAudioElement | null>(null)

  const handleUpload = async (file: File) => {
    const formData = new FormData()
    formData.append('file', file)

    // If we already have a course ID, send it so the file gets added to that course
    if (courseId) {
      formData.append('course_id', courseId)
    }

    try {
      setUploadLoading(true)
      const response = await axios.post(`${API_BASE}/upload`, formData)

      // If this is the first file, set the course ID
      if (!courseId) {
        setCourseId(response.data.course_id)
      }

      // Fetch updated file list from the API
      const filesResponse = await axios.get(`${API_BASE}/files/${courseId || response.data.course_id}`)
      setSourceFiles(filesResponse.data.files || [])

      message.success(`${response.data.pdf_name} uploaded successfully!`)
    } catch (error) {
      console.error('Upload error:', error)
      message.error('Failed to upload PDF')
    } finally {
      setUploadLoading(false)
    }
    return false
  }

  const handleDeleteFile = async (fileId: string) => {
    if (!courseId) return

    try {
      const response = await axios.delete(`${API_BASE}/files/${courseId}/${fileId}`)

      // Refresh file list
      const filesResponse = await axios.get(`${API_BASE}/files/${courseId}`)
      setSourceFiles(filesResponse.data.files || [])

      message.success('File deleted successfully')

      // If all files are removed, reset the course
      if (response.data.files_remaining === 0) {
        setCourseId(null)
        setSlides([])
        setMessages([])
        setQuestions([])
        setQuizAnswers([])
      }
    } catch (error) {
      console.error('Delete error:', error)
      message.error('Failed to delete file')
    }
  }

  const handleGenerateCourse = async () => {
    if (!courseId) {
      message.error('Please upload at least one file first')
      return
    }

    try {
      setGenLoading(true)
      setGenModalVisible(false) // Close modal when generation starts

      try {
        await axios.post(`${API_BASE}/course/generate`, {
          course_id: courseId,
          num_slides: numSlides,
          presentation_style: presentationStyle,
          instructor_prompt: instructorPrompt,
          generate_images: generateImages,
          use_web_images: useWebImages,
          use_dalle: useDalle,
          generate_voiceover: generateVoiceover,
          generate_questions: generateQuestions,
          language: language,
        }, {
          timeout: 300000, // 5 minutes timeout to match nginx
        })
      } catch (genError) {
        // If timeout or network error, the course might still be generated
        // Let's try to fetch the slides anyway after a short delay
        console.log('Generation request failed/timed out, checking for results...')
        await new Promise(resolve => setTimeout(resolve, 3000))
      }

      // Fetch the slides (they may have been generated even if request timed out)
      const slidesResponse = await axios.get(`${API_BASE}/slides/${courseId}`)

      if (slidesResponse.data.slides && slidesResponse.data.slides.length > 0) {
        setSlides(slidesResponse.data.slides)
        setCurrentSlide(0)

        // Fetch questions if they were generated
        if (generateQuestions) {
          const questionsResponse = await axios.get(`${API_BASE}/questions/${courseId}`)
          setQuestions(questionsResponse.data.questions || [])
          setQuizAnswers([])
        }

        message.success(`Course generated with ${slidesResponse.data.slides.length} slides!`)
      } else {
        throw new Error('No slides were generated')
      }
    } catch (error) {
      console.error('Generation error:', error)
      message.error('Failed to generate course')
    } finally {
      setGenLoading(false)
    }
  }

  const handleAsk = async () => {
    if (!question.trim() || !courseId) return

    const userMsg: ChatMessage = { role: 'user', content: question }
    setMessages((prev) => [...prev, userMsg])
    setQuestion('')

    try {
      setLoading(true)
      const response = await axios.post(`${API_BASE}/chat/ask`, {
        course_id: courseId,
        question: question,
      })

      const assistantMsg: ChatMessage = {
        role: 'assistant',
        content: response.data.answer,
        citations: response.data.citations,
      }
      setMessages((prev) => [...prev, assistantMsg])
      setTimeout(() => chatEndRef.current?.scrollIntoView({ behavior: 'smooth' }), 100)
    } catch (error) {
      console.error('Chat error:', error)
      message.error('Failed to get answer')
    } finally {
      setLoading(false)
    }
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {/* Header */}
      <Header
        style={{
          background: darkMode ? '#001529' : '#fff',
          padding: '12px 24px',
          boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <BookOutlined style={{ fontSize: '24px', color: '#1890ff' }} />
          <Title level={4} style={{ margin: 0, color: darkMode ? '#fff' : '#000' }}>
            AI eLearning
          </Title>
        </div>
        <Tooltip title={darkMode ? 'Light Mode' : 'Dark Mode'}>
          <Switch
            checkedChildren={<MoonOutlined />}
            unCheckedChildren={<SunOutlined />}
            checked={darkMode}
            onChange={setDarkMode}
          />
        </Tooltip>
      </Header>

      <Layout>
        {/* Left Sidebar - File Management */}
        <Sider
          width={280}
          theme={darkMode ? 'dark' : 'light'}
          style={{
            background: darkMode ? '#001529' : '#fff',
            borderRight: `1px solid ${darkMode ? '#303030' : '#f0f0f0'}`,
          }}
        >
          <div style={{ padding: '16px', height: '100%', display: 'flex', flexDirection: 'column' }}>
            <Title level={5} style={{ color: darkMode ? '#fff' : '#000' }}>Source Files</Title>
            <Divider style={{ margin: '12px 0' }} />

            {/* File Upload */}
            <Upload.Dragger
              beforeUpload={handleUpload}
              showUploadList={false}
              accept=".pdf"
              disabled={uploadLoading}
              style={{ marginBottom: '16px' }}
            >
              <p className="ant-upload-drag-icon">
                <UploadOutlined style={{ fontSize: '32px', color: '#1890ff' }} />
              </p>
              <p className="ant-upload-text" style={{ fontSize: '14px' }}>
                {uploadLoading ? 'Processing...' : 'Drop PDF here or click'}
              </p>
            </Upload.Dragger>

            {/* File List */}
            <div style={{ flex: 1, overflowY: 'auto', marginBottom: '16px' }}>
              {sourceFiles.length === 0 ? (
                <Empty
                  description="No files uploaded"
                  image={Empty.PRESENTED_IMAGE_SIMPLE}
                />
              ) : (
                <List
                  size="small"
                  dataSource={sourceFiles}
                  renderItem={(file) => (
                    <List.Item
                      actions={[
                        <Button
                          type="text"
                          danger
                          size="small"
                          icon={<DeleteOutlined />}
                          onClick={() => handleDeleteFile(file.id)}
                        />
                      ]}
                    >
                      <List.Item.Meta
                        avatar={<FileTextOutlined style={{ fontSize: '16px' }} />}
                        title={<Text style={{ fontSize: '12px' }}>{file.filename}</Text>}
                      />
                    </List.Item>
                  )}
                />
              )}
            </div>

            {/* Generate Button */}
            <Button
              type="primary"
              size="large"
              block
              icon={<BulbOutlined />}
              onClick={() => setGenModalVisible(true)}
              disabled={sourceFiles.length === 0}
              loading={genLoading}
            >
              {slides.length > 0 ? 'Regenerate Course' : 'Generate Course'}
            </Button>

            {/* Course Info */}
            {slides.length > 0 && (
              <Card size="small" style={{ marginTop: '16px' }}>
                <Text type="secondary" style={{ fontSize: '12px' }}>
                  {slides.length} slides generated
                </Text>
              </Card>
            )}
          </div>
        </Sider>

        {/* Main Content */}
        <Content style={{ display: 'flex', flexDirection: 'row', height: 'calc(100vh - 64px)' }}>
          {/* Center - Slide Viewer */}
          <div style={{ flex: 1, padding: '24px', overflowY: 'auto', background: darkMode ? '#141414' : '#f0f2f5' }}>
            {slides.length === 0 ? (
              <Card style={{ textAlign: 'center', padding: '48px', height: '100%' }}>
                <Empty
                  description={
                    <div>
                      <Title level={3}>Welcome to AI eLearning</Title>
                      <Paragraph type="secondary">
                        Upload PDF files on the left to get started, then click "Generate Course" to create an interactive learning experience with AI.
                      </Paragraph>
                    </div>
                  }
                  image={Empty.PRESENTED_IMAGE_DEFAULT}
                />
              </Card>
            ) : (
              <>
                {/* Slide Navigation */}
                <Card
                  title={
                    <Space>
                      <FileTextOutlined />
                      <Text strong>Slide {currentSlide + 1} of {slides.length}</Text>
                    </Space>
                  }
                  extra={
                    <Space wrap>
                      {slides[currentSlide]?.audio_url && (
                        <Button
                          icon={<SoundOutlined />}
                          onClick={() => {
                            if (audioRef.current) {
                              audioRef.current.pause()
                              audioRef.current = null
                            }

                            const baseUrl = import.meta.env.DEV ? 'http://localhost:8080' : ''
                            const audio = new Audio(`${baseUrl}${slides[currentSlide].audio_url}`)
                            audioRef.current = audio
                            audio.play()

                            audio.onended = () => {
                              if (currentSlide < slides.length - 1) {
                                setCurrentSlide(currentSlide + 1)
                              }
                              audioRef.current = null
                            }
                          }}
                        >
                          Present
                        </Button>
                      )}
                      {slides[currentSlide]?.instructor_script && (
                        <Button
                          icon={<ReadOutlined />}
                          onClick={() => setNotesVisible(true)}
                        >
                          Script
                        </Button>
                      )}
                      <Button
                        icon={<UnorderedListOutlined />}
                        onClick={() => setTocVisible(true)}
                      >
                        Contents
                      </Button>
                      <Button
                        icon={<LeftOutlined />}
                        disabled={currentSlide === 0}
                        onClick={() => {
                          if (audioRef.current) {
                            audioRef.current.pause()
                            audioRef.current = null
                          }
                          setCurrentSlide(currentSlide - 1)
                        }}
                      >
                        Previous
                      </Button>
                      <Button
                        type="primary"
                        icon={<RightOutlined />}
                        iconPosition="end"
                        disabled={currentSlide === slides.length - 1}
                        onClick={() => {
                          if (audioRef.current) {
                            audioRef.current.pause()
                            audioRef.current = null
                          }
                          setCurrentSlide(currentSlide + 1)
                        }}
                      >
                        Next
                      </Button>
                    </Space>
                  }
                  style={{ marginBottom: '24px' }}
                  styles={{ body: { padding: 0 } }}
                >
                  {(() => {
                    const slide = slides[currentSlide]
                    const themeColors = {
                      blue: { bg: '#e6f7ff', border: '#1890ff', text: '#001529', accent: '#40a9ff' },
                      green: { bg: '#f6ffed', border: '#52c41a', text: '#135200', accent: '#73d13d' },
                      purple: { bg: '#f9f0ff', border: '#722ed1', text: '#22075e', accent: '#9254de' },
                      orange: { bg: '#fff7e6', border: '#fa8c16', text: '#ad4e00', accent: '#ffa940' },
                      gradient: { bg: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: '#667eea', text: '#fff', accent: '#a78bfa' },
                    }
                    const theme = themeColors[slide?.theme as keyof typeof themeColors] || themeColors.blue
                    const isGradient = slide?.theme === 'gradient'
                    const layout = slide?.layout || 'standard'

                    // Render different layouts based on the layout type
                    const renderSlideContent = () => {
                      // Title slide - full screen with centered content and background image
                      if (layout === 'title') {
                        return (
                          <div
                            style={{
                              background: slide?.image_url
                                ? `linear-gradient(rgba(0,0,0,0.5), rgba(0,0,0,0.5)), url(${slide.image_url}) center/cover`
                                : theme.bg,
                              minHeight: '500px',
                              padding: '80px 48px',
                              display: 'flex',
                              flexDirection: 'column',
                              justifyContent: 'center',
                              alignItems: 'center',
                              textAlign: 'center',
                              position: 'relative',
                              borderLeft: `4px solid ${theme.border}`,
                            }}
                          >
                            <Title level={1} style={{ color: slide?.image_url ? '#fff' : (isGradient ? '#fff' : theme.text), marginBottom: '24px', fontSize: '48px', textShadow: slide?.image_url ? '2px 2px 4px rgba(0,0,0,0.7)' : 'none' }}>
                              {slide.title}
                            </Title>
                            {slide.content && (
                              <Text style={{ fontSize: '20px', color: slide?.image_url ? '#fff' : (isGradient ? '#fff' : theme.text), maxWidth: '800px', textShadow: slide?.image_url ? '1px 1px 2px rgba(0,0,0,0.7)' : 'none' }}>
                                {slide.content}
                              </Text>
                            )}
                          </div>
                        )
                      }

                      // Split layout - image on left, content on right (50/50)
                      if (layout === 'split' || layout === 'comparison') {
                        return (
                          <div style={{ background: theme.bg, minHeight: '450px', display: 'flex', borderLeft: `4px solid ${theme.border}` }}>
                            {slide?.image_url && (
                              <div style={{ flex: 1, background: `url(${slide.image_url}) center/cover`, minHeight: '450px' }} />
                            )}
                            <div style={{ flex: 1, padding: '32px', display: 'flex', flexDirection: 'column', justifyContent: 'center' }}>
                              <Title level={2} style={{ color: isGradient ? '#fff' : theme.text, marginBottom: '16px' }}>
                                {slide.title}
                              </Title>
                              <Divider style={{ borderColor: theme.border, opacity: 0.5 }} />
                              <Paragraph style={{ fontSize: '16px', lineHeight: '1.8', whiteSpace: 'pre-line', color: isGradient ? '#fff' : theme.text }}>
                                {slide.content}
                              </Paragraph>
                            </div>
                          </div>
                        )
                      }

                      // List layout - content with sidebar image
                      if (layout === 'list') {
                        return (
                          <div style={{ background: theme.bg, minHeight: '450px', padding: '32px', borderLeft: `4px solid ${theme.border}` }}>
                            <div style={{ display: 'flex', gap: '32px', alignItems: 'start' }}>
                              <div style={{ flex: 2 }}>
                                <Title level={2} style={{ color: isGradient ? '#fff' : theme.text, marginBottom: '16px' }}>
                                  {slide.title}
                                </Title>
                                <Divider style={{ borderColor: theme.border, opacity: 0.5 }} />
                                <div style={{ fontSize: '16px', lineHeight: '1.8', whiteSpace: 'pre-line', color: isGradient ? '#fff' : theme.text }}>
                                  {slide.content.split('\n').map((line, idx) => {
                                    if (line.trim().startsWith('-') || line.trim().startsWith('•')) {
                                      return (
                                        <div key={idx} style={{ display: 'flex', gap: '12px', marginBottom: '12px', alignItems: 'start' }}>
                                          <span style={{ color: theme.accent, fontSize: '20px', fontWeight: 'bold', minWidth: '20px' }}>•</span>
                                          <span style={{ flex: 1 }}>{line.replace(/^[-•]\s*/, '')}</span>
                                        </div>
                                      )
                                    }
                                    return <div key={idx} style={{ marginBottom: '8px' }}>{line}</div>
                                  })}
                                </div>
                              </div>
                              {slide?.image_url && (
                                <div style={{ flex: 1, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                                  <img
                                    src={slide.image_url}
                                    alt={slide.image_prompt || slide.title}
                                    style={{ maxWidth: '100%', borderRadius: '12px', boxShadow: '0 8px 24px rgba(0,0,0,0.15)' }}
                                  />
                                </div>
                              )}
                            </div>
                          </div>
                        )
                      }

                      // Summary/concept layout - large image on top with overlay text box
                      if (layout === 'summary' || layout === 'concept') {
                        return (
                          <div style={{ background: theme.bg, minHeight: '450px', position: 'relative', borderLeft: `4px solid ${theme.border}` }}>
                            {slide?.image_url && (
                              <div
                                style={{
                                  width: '100%',
                                  height: '250px',
                                  background: `url(${slide.image_url}) center/cover`,
                                  borderBottom: `4px solid ${theme.border}`
                                }}
                              />
                            )}
                            <div style={{ padding: '32px' }}>
                              <Card
                                style={{
                                  background: 'rgba(255,255,255,0.95)',
                                  marginTop: slide?.image_url ? '-60px' : '0',
                                  position: 'relative',
                                  borderLeft: `4px solid ${theme.accent}`,
                                  boxShadow: '0 4px 12px rgba(0,0,0,0.1)'
                                }}
                              >
                                <Title level={2} style={{ color: theme.text, marginTop: 0, marginBottom: '16px' }}>
                                  {slide.title}
                                </Title>
                                <Paragraph style={{ fontSize: '16px', lineHeight: '1.8', whiteSpace: 'pre-line', color: theme.text, marginBottom: 0 }}>
                                  {slide.content}
                                </Paragraph>
                              </Card>
                            </div>
                          </div>
                        )
                      }

                      // Data/highlight layout - image as small icon, big text
                      if (layout === 'data' || layout === 'highlight') {
                        return (
                          <div style={{ background: theme.bg, minHeight: '450px', padding: '48px', borderLeft: `4px solid ${theme.border}`, textAlign: 'center' }}>
                            {slide?.image_url && (
                              <div style={{ marginBottom: '24px' }}>
                                <img
                                  src={slide.image_url}
                                  alt={slide.image_prompt || slide.title}
                                  style={{ width: '120px', height: '120px', borderRadius: '50%', objectFit: 'cover', border: `4px solid ${theme.border}`, boxShadow: '0 4px 12px rgba(0,0,0,0.15)' }}
                                />
                              </div>
                            )}
                            <Title level={1} style={{ color: isGradient ? '#fff' : theme.text, marginBottom: '24px', fontSize: '36px' }}>
                              {slide.title}
                            </Title>
                            <div style={{ maxWidth: '700px', margin: '0 auto' }}>
                              <Paragraph style={{ fontSize: '18px', lineHeight: '1.8', whiteSpace: 'pre-line', color: isGradient ? '#fff' : theme.text }}>
                                {slide.content}
                              </Paragraph>
                            </div>
                          </div>
                        )
                      }

                      // Default/standard layout - traditional slide with image on top
                      return (
                        <div style={{ background: theme.bg, minHeight: '450px', padding: '32px', borderLeft: `4px solid ${theme.border}` }}>
                          {slide?.image_url && (
                            <div style={{ marginBottom: '24px', textAlign: 'center' }}>
                              <img
                                src={slide.image_url}
                                alt={slide.image_prompt || slide.title}
                                style={{ maxWidth: '100%', maxHeight: '300px', borderRadius: '8px', boxShadow: '0 4px 12px rgba(0,0,0,0.15)' }}
                              />
                            </div>
                          )}
                          <Title level={2} style={{ color: isGradient ? '#fff' : theme.text, marginBottom: '16px' }}>
                            {slide.title}
                          </Title>
                          <Divider style={{ borderColor: theme.border, opacity: 0.5 }} />
                          <Paragraph style={{ fontSize: '16px', lineHeight: '1.8', whiteSpace: 'pre-line', color: isGradient ? '#fff' : theme.text }}>
                            {slide.content}
                          </Paragraph>
                        </div>
                      )
                    }

                    return renderSlideContent()
                  })()}
                  <div style={{ padding: '16px 24px' }}>
                    <Progress
                      percent={Math.round(((currentSlide + 1) / slides.length) * 100)}
                      strokeColor={{
                        '0%': '#108ee9',
                        '100%': '#87d068',
                      }}
                    />
                  </div>
                </Card>

                {/* Quiz Component */}
                {questions.length > 0 && questions[currentSlide] && (
                  <Card
                    title={
                      <Space>
                        <BulbOutlined />
                        <Text strong>Quiz Question</Text>
                      </Space>
                    }
                  >
                    <Paragraph style={{ fontSize: '16px', fontWeight: 500, marginBottom: '16px' }}>
                      {questions[currentSlide].question}
                    </Paragraph>
                    <Space direction="vertical" style={{ width: '100%' }}>
                      {questions[currentSlide].options.map((option, idx) => {
                        const slideAnswer = quizAnswers.find(a => a.slideId === slides[currentSlide].id)
                        const isSelected = slideAnswer?.selectedAnswer === idx
                        const isCorrect = questions[currentSlide].correct_answer === idx
                        const showResult = slideAnswer !== undefined

                        let buttonStyle: React.CSSProperties = {
                          width: '100%',
                          textAlign: 'left',
                          height: 'auto',
                          padding: '12px',
                        }

                        if (showResult) {
                          if (isCorrect) {
                            buttonStyle = { ...buttonStyle, backgroundColor: '#52c41a', color: '#fff', borderColor: '#52c41a' }
                          } else if (isSelected) {
                            buttonStyle = { ...buttonStyle, backgroundColor: '#ff4d4f', color: '#fff', borderColor: '#ff4d4f' }
                          }
                        }

                        return (
                          <Button
                            key={idx}
                            style={buttonStyle}
                            disabled={showResult}
                            onClick={() => {
                              const isCorrectAnswer = idx === questions[currentSlide].correct_answer
                              setQuizAnswers([
                                ...quizAnswers.filter(a => a.slideId !== slides[currentSlide].id),
                                {
                                  slideId: slides[currentSlide].id,
                                  selectedAnswer: idx,
                                  isCorrect: isCorrectAnswer
                                }
                              ])
                            }}
                          >
                            {String.fromCharCode(65 + idx)}. {option}
                          </Button>
                        )
                      })}
                    </Space>
                  </Card>
                )}
              </>
            )}
          </div>

          {/* Right Sidebar - Chat */}
          <div style={{ width: '400px', borderLeft: `1px solid ${darkMode ? '#303030' : '#f0f0f0'}` }}>
            <Card
              title={
                <Space>
                  <RobotOutlined />
                  <Text strong>AI Assistant</Text>
                  <Badge count={messages.length} size="small" />
                </Space>
              }
              style={{ height: '100%', display: 'flex', flexDirection: 'column' }}
              styles={{ body: { flex: 1, display: 'flex', flexDirection: 'column', padding: 0 } }}
            >
              <div
                style={{
                  flex: 1,
                  overflowY: 'auto',
                  padding: '16px',
                  background: darkMode ? '#1f1f1f' : '#fafafa',
                }}
              >
                {messages.length === 0 ? (
                  <Empty
                    description="Ask me anything about the course!"
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                  />
                ) : (
                  <List
                    dataSource={messages}
                    renderItem={(msg) => (
                      <List.Item style={{ border: 'none', padding: '8px 0' }}>
                        <Card
                          size="small"
                          style={{
                            width: '100%',
                            background: msg.role === 'user' ? '#e6f7ff' : '#fff',
                            borderLeft: `3px solid ${msg.role === 'user' ? '#1890ff' : '#52c41a'}`,
                          }}
                        >
                          <div style={{ display: 'flex', gap: '12px' }}>
                            <Avatar
                              icon={msg.role === 'user' ? <UserOutlined /> : <RobotOutlined />}
                              style={{
                                background: msg.role === 'user' ? '#1890ff' : '#52c41a',
                              }}
                            />
                            <div style={{ flex: 1 }}>
                              <Text strong>{msg.role === 'user' ? 'You' : 'AI'}</Text>
                              <Paragraph style={{ marginTop: '8px', marginBottom: 0 }}>
                                {msg.content}
                              </Paragraph>
                              {msg.citations && msg.citations.length > 0 && (
                                <div style={{ marginTop: '8px' }}>
                                  <Text type="secondary" style={{ fontSize: '12px' }}>Sources: </Text>
                                  {msg.citations.map((cite, i) => (
                                    <Tag key={i} color="blue" style={{ fontSize: '11px' }}>
                                      {cite}
                                    </Tag>
                                  ))}
                                </div>
                              )}
                            </div>
                          </div>
                        </Card>
                      </List.Item>
                    )}
                  />
                )}
                <div ref={chatEndRef} />
              </div>
              <div style={{ padding: '16px', borderTop: '1px solid #f0f0f0' }}>
                <Space.Compact style={{ width: '100%' }}>
                  <Input
                    placeholder="Ask a question..."
                    value={question}
                    onChange={(e) => setQuestion(e.target.value)}
                    onPressEnter={handleAsk}
                    disabled={loading || !courseId}
                  />
                  <Button
                    type="primary"
                    icon={<SendOutlined />}
                    onClick={handleAsk}
                    loading={loading}
                    disabled={!question.trim() || !courseId}
                  >
                    Send
                  </Button>
                </Space.Compact>
              </div>
            </Card>
          </div>
        </Content>
      </Layout>

      {/* Generation Settings Modal */}
      <Modal
        title={
          <Space>
            <BulbOutlined />
            <Text strong>Course Generation Settings</Text>
          </Space>
        }
        open={genModalVisible}
        onCancel={() => setGenModalVisible(false)}
        onOk={handleGenerateCourse}
        okText={slides.length > 0 ? 'Regenerate Course' : 'Generate Course'}
        width={600}
        confirmLoading={genLoading}
      >
        <Space direction="vertical" size="large" style={{ width: '100%' }}>
          <div>
            <Text strong>Number of Slides:</Text>
            <InputNumber
              min={3}
              max={50}
              value={numSlides}
              onChange={(val) => setNumSlides(val || 10)}
              size="large"
              style={{ width: '100%', marginTop: '8px' }}
            />
          </div>

          <div>
            <Text strong>Presentation Style:</Text>
            <Select
              value={presentationStyle}
              onChange={setPresentationStyle}
              size="large"
              style={{ width: '100%', marginTop: '8px' }}
              options={[
                {
                  value: 'minimal',
                  label: 'Minimal & Visual',
                },
                {
                  value: 'balanced',
                  label: 'Balanced',
                },
                {
                  value: 'detailed',
                  label: 'Detailed & Professional',
                },
                {
                  value: 'fun',
                  label: 'Fun & Entertaining',
                },
              ]}
            />
            <Text type="secondary" style={{ fontSize: '12px', display: 'block', marginTop: '8px' }}>
              {presentationStyle === 'minimal' && '✨ Minimal text, large visuals, modern design with bold statements'}
              {presentationStyle === 'balanced' && '📊 Balanced mix of text and visuals for general audiences'}
              {presentationStyle === 'detailed' && '📝 Comprehensive text, professional layout, detailed explanations'}
              {presentationStyle === 'fun' && '🎉 Engaging and entertaining with minimal text, playful imagery'}
            </Text>
          </div>

          <div>
            <Text strong>Content Language:</Text>
            <Select
              value={language}
              onChange={setLanguage}
              size="large"
              style={{ width: '100%', marginTop: '8px' }}
              options={[
                { value: 'english', label: 'English' },
                { value: 'indonesian', label: 'Indonesian (Bahasa Indonesia)' },
                { value: 'thai', label: 'Thai (ภาษาไทย)' },
                { value: 'german', label: 'German (Deutsch)' },
              ]}
            />
          </div>

          <div>
            <Text strong>Instructor Presentation Style:</Text>
            <TextArea
              value={instructorPrompt}
              onChange={(e) => setInstructorPrompt(e.target.value)}
              placeholder="Describe the presentation style..."
              autoSize={{ minRows: 2, maxRows: 4 }}
              style={{ marginTop: '8px' }}
            />
          </div>

          <div>
            <Text strong>Image Options:</Text>
            <div style={{ marginTop: '8px', marginLeft: '16px' }}>
              <Checkbox
                checked={generateImages}
                onChange={(e) => {
                  setGenerateImages(e.target.checked)
                  if (!e.target.checked) {
                    setUseWebImages(false)
                    setUseDalle(false)
                  }
                }}
              >
                <Text strong>Add Images to Slides</Text>
              </Checkbox>

              {generateImages && (
                <div style={{ marginLeft: '24px', marginTop: '8px' }}>
                  <Checkbox
                    checked={useWebImages}
                    onChange={(e) => setUseWebImages(e.target.checked)}
                  >
                    Use Professional Stock Photos (Faster, Free)
                  </Checkbox>
                  <br />
                  <Checkbox
                    checked={useDalle}
                    onChange={(e) => setUseDalle(e.target.checked)}
                    style={{ marginTop: '8px' }}
                  >
                    Use AI-Generated Images (DALL-E 3, Slower)
                  </Checkbox>
                  <br />
                  <Text type="secondary" style={{ fontSize: '12px', marginLeft: '24px' }}>
                    If both are selected, stock photos will be tried first, with AI as fallback
                  </Text>
                </div>
              )}
            </div>
          </div>

          <div>
            <Checkbox
              checked={generateVoiceover}
              onChange={(e) => setGenerateVoiceover(e.target.checked)}
            >
              Generate Voiceover (Text-to-Speech)
            </Checkbox>
          </div>

          <div>
            <Checkbox
              checked={generateQuestions}
              onChange={(e) => setGenerateQuestions(e.target.checked)}
            >
              Generate Quiz Questions
            </Checkbox>
          </div>

          {genLoading && (
            <div>
              <Progress percent={50} status="active" />
              <Text type="secondary">Generating course content...</Text>
            </div>
          )}
        </Space>
      </Modal>

      {/* Instructor Script Modal */}
      <Modal
        title={
          <Space>
            <ReadOutlined />
            <Text strong>Presentation Script - Slide {currentSlide + 1}</Text>
          </Space>
        }
        open={notesVisible}
        onCancel={() => setNotesVisible(false)}
        footer={[
          <Button key="close" type="primary" onClick={() => setNotesVisible(false)}>
            Close
          </Button>
        ]}
        width={700}
      >
        <Card style={{ background: '#f0f2f5', border: 'none' }}>
          <Title level={5} style={{ marginTop: 0 }}>What to say when presenting this slide:</Title>
          <Paragraph style={{ fontSize: '16px', lineHeight: '1.8', margin: 0, whiteSpace: 'pre-line' }}>
            {slides[currentSlide]?.instructor_script || 'No presentation script available for this slide.'}
          </Paragraph>
        </Card>
      </Modal>

      {/* Table of Contents Modal */}
      <Modal
        title={
          <Space>
            <UnorderedListOutlined />
            <Text strong>Table of Contents</Text>
          </Space>
        }
        open={tocVisible}
        onCancel={() => setTocVisible(false)}
        footer={[
          <Button key="close" onClick={() => setTocVisible(false)}>
            Close
          </Button>
        ]}
        width={600}
      >
        <List
          dataSource={slides}
          renderItem={(slide, idx) => (
            <List.Item
              style={{
                cursor: 'pointer',
                padding: '12px 16px',
                background: idx === currentSlide ? '#e6f7ff' : 'transparent',
                borderLeft: idx === currentSlide ? '3px solid #1890ff' : '3px solid transparent',
                transition: 'all 0.3s',
              }}
              onClick={() => {
                setCurrentSlide(idx)
                setTocVisible(false)
                if (audioRef.current) {
                  audioRef.current.pause()
                  audioRef.current = null
                }
              }}
              onMouseEnter={(e) => {
                if (idx !== currentSlide) {
                  e.currentTarget.style.background = '#f0f0f0'
                }
              }}
              onMouseLeave={(e) => {
                if (idx !== currentSlide) {
                  e.currentTarget.style.background = 'transparent'
                }
              }}
            >
              <Space>
                <Badge count={idx + 1} style={{ backgroundColor: '#1890ff' }} />
                <Text strong>{slide.title}</Text>
              </Space>
            </List.Item>
          )}
        />
      </Modal>
    </Layout>
  )
}

export default App
