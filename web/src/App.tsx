import { useState, useRef } from 'react'
import {
  Layout,
  Upload,
  Button,
  Card,
  Steps,
  InputNumber,
  Progress,
  Input,
  List,
  Avatar,
  Tag,
  Menu,
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
} from 'antd'
import {
  UploadOutlined,
  FileTextOutlined,
  SendOutlined,
  LeftOutlined,
  RightOutlined,
  MessageOutlined,
  BookOutlined,
  RobotOutlined,
  UserOutlined,
  BulbOutlined,
  MoonOutlined,
  SunOutlined,
  ReadOutlined,
  SoundOutlined,
} from '@ant-design/icons'
import axios from 'axios'

const { Header, Content, Sider } = Layout
const { Title, Text, Paragraph } = Typography
const { TextArea } = Input

const API_BASE = import.meta.env.VITE_API_URL || (import.meta.env.DEV ? 'http://localhost:8080/api' : '/api')

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
  const [pdfName, setPdfName] = useState('')
  const [slides, setSlides] = useState<Slide[]>([])
  const [currentSlide, setCurrentSlide] = useState(0)
  const [messages, setMessages] = useState<ChatMessage[]>([])
  const [question, setQuestion] = useState('')
  const [loading, setLoading] = useState(false)
  const [uploadLoading, setUploadLoading] = useState(false)
  const [genLoading, setGenLoading] = useState(false)
  const [numSlides, setNumSlides] = useState(10)
  const [instructorPrompt, setInstructorPrompt] = useState('friendly, conversational, engaging presentation style targeted at a general audience. Do not ask questions at the end of slides.')
  const [generateImages, setGenerateImages] = useState(false)
  const [generateVoiceover, setGenerateVoiceover] = useState(false)
  const [generateQuestions, setGenerateQuestions] = useState(false)
  const [chatOpen, setChatOpen] = useState(true)
  const [notesVisible, setNotesVisible] = useState(false)
  const [questions, setQuestions] = useState<Question[]>([])
  const [quizAnswers, setQuizAnswers] = useState<QuizAnswer[]>([])
  const chatEndRef = useRef<HTMLDivElement>(null)
  const audioRef = useRef<HTMLAudioElement | null>(null)

  const handleUpload = async (file: File) => {
    const formData = new FormData()
    formData.append('file', file)

    try {
      setUploadLoading(true)
      const response = await axios.post(`${API_BASE}/upload`, formData)
      setCourseId(response.data.course_id)
      setPdfName(response.data.pdf_name)
      message.success('PDF uploaded and processed successfully!')
    } catch (error) {
      console.error('Upload error:', error)
      message.error('Failed to upload PDF')
    } finally {
      setUploadLoading(false)
    }
    return false
  }

  const handleGenerateCourse = async () => {
    if (!courseId) return

    try {
      setGenLoading(true)
      await axios.post(`${API_BASE}/course/generate`, {
        course_id: courseId,
        num_slides: numSlides,
        instructor_prompt: instructorPrompt,
        generate_images: generateImages,
        generate_voiceover: generateVoiceover,
        generate_questions: generateQuestions,
      })

      const slidesResponse = await axios.get(`${API_BASE}/slides/${courseId}`)
      setSlides(slidesResponse.data.slides || [])
      setCurrentSlide(0)

      // Fetch questions if they were generated
      if (generateQuestions) {
        const questionsResponse = await axios.get(`${API_BASE}/questions/${courseId}`)
        setQuestions(questionsResponse.data.questions || [])
        setQuizAnswers([])
      }

      message.success(`Course generated with ${slidesResponse.data.slides.length} slides!`)
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

  const getCurrentStep = () => {
    if (!courseId) return 0
    if (slides.length === 0) return 1
    return 2
  }

  return (
    <Layout style={{ minHeight: '100vh' }}>
      {/* Header */}
      <Header
        style={{
          background: darkMode ? '#001529' : '#fff',
          padding: '12px 16px',
          boxShadow: '0 2px 8px rgba(0,0,0,0.1)',
        }}
      >
        <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', flexWrap: 'wrap', gap: '12px' }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
            <BookOutlined style={{ fontSize: '24px', color: '#1890ff' }} />
            <Title level={4} style={{ margin: 0, color: darkMode ? '#fff' : '#000', fontSize: '18px' }}>
              AI eLearning
            </Title>
          </div>
          <Space size="small" wrap>
            <div style={{ display: window.innerWidth > 768 ? 'block' : 'none' }}>
              <Steps
                current={getCurrentStep()}
                size="small"
                items={[
                  { title: 'Upload', icon: <UploadOutlined /> },
                  { title: 'Generate', icon: <BookOutlined /> },
                  { title: 'Learn', icon: <MessageOutlined /> },
                ]}
                style={{ minWidth: '300px' }}
              />
            </div>
            <Tooltip title={darkMode ? 'Light Mode' : 'Dark Mode'}>
              <Switch
                checkedChildren={<MoonOutlined />}
                unCheckedChildren={<SunOutlined />}
                checked={darkMode}
                onChange={setDarkMode}
                size="small"
              />
            </Tooltip>
            {slides.length > 0 && (
              <Button
                type="primary"
                icon={<MessageOutlined />}
                onClick={() => setChatOpen(!chatOpen)}
                size="small"
              >
                <span style={{ display: window.innerWidth > 768 ? 'inline' : 'none' }}>
                  {chatOpen ? 'Hide Chat' : 'Show Chat'}
                </span>
              </Button>
            )}
          </Space>
        </div>
      </Header>

      <Layout>
        {/* Sidebar - Course Navigation */}
        {slides.length > 0 && (
          <Sider
            width={250}
            theme={darkMode ? 'dark' : 'light'}
            breakpoint="lg"
            collapsedWidth="0"
          >
            <div style={{ padding: '16px' }}>
              <Title level={5}>Course Outline</Title>
              <Divider style={{ margin: '12px 0' }} />
              <Menu
                mode="inline"
                selectedKeys={[String(currentSlide)]}
                items={slides.map((slide, idx) => ({
                  key: String(idx),
                  icon: <FileTextOutlined />,
                  label: `${idx + 1}. ${slide.title}`,
                  onClick: () => {
                    if (audioRef.current) {
                      audioRef.current.pause()
                      audioRef.current = null
                    }
                    setCurrentSlide(idx)
                  },
                }))}
              />
            </div>
          </Sider>
        )}

        {/* Main Content */}
        <Content style={{ padding: window.innerWidth > 768 ? '24px' : '12px', background: darkMode ? '#141414' : '#f0f2f5' }}>
          {/* Upload Section */}
          {!courseId && (
            <Card style={{ textAlign: 'center', padding: window.innerWidth > 768 ? '48px' : '24px' }}>
              <Space direction="vertical" size="large" style={{ width: '100%' }}>
                <UploadOutlined style={{ fontSize: '64px', color: '#1890ff' }} />
                <Title level={2}>Upload Your PDF Document</Title>
                <Text type="secondary">
                  Upload a PDF to create an AI-powered interactive course with slides and chatbot
                </Text>
                <Upload.Dragger
                  beforeUpload={handleUpload}
                  showUploadList={false}
                  accept=".pdf"
                  disabled={uploadLoading}
                  style={{ padding: '40px' }}
                >
                  <p className="ant-upload-drag-icon">
                    <UploadOutlined style={{ fontSize: '48px', color: '#1890ff' }} />
                  </p>
                  <p className="ant-upload-text" style={{ fontSize: '18px', fontWeight: 'bold' }}>
                    {uploadLoading ? 'Processing PDF...' : 'Click or drag PDF file here to upload'}
                  </p>
                  <p className="ant-upload-hint" style={{ fontSize: '14px' }}>
                    Supports PDF files up to 50MB
                  </p>
                </Upload.Dragger>
              </Space>
            </Card>
          )}

          {/* Generate Course Section */}
          {courseId && slides.length === 0 && (
            <Card title={`Generate Course from: ${pdfName}`}>
              <Space direction="vertical" size="large" style={{ width: '100%' }}>
                <div>
                  <Text strong>Instructor Presentation Style:</Text>
                  <br />
                  <TextArea
                    value={instructorPrompt}
                    onChange={(e) => setInstructorPrompt(e.target.value)}
                    placeholder="Describe the presentation style for the instructor script..."
                    autoSize={{ minRows: 2, maxRows: 4 }}
                    style={{ marginTop: '8px' }}
                  />
                  <Text type="secondary">
                    <br />
                    AI will generate a complete presentation script for each slide matching this style
                  </Text>
                </div>
                <div>
                  <Text strong>Number of Slides:</Text>
                  <br />
                  <InputNumber
                    min={3}
                    max={50}
                    value={numSlides}
                    onChange={(val) => setNumSlides(val || 10)}
                    size="large"
                    style={{ width: '100%', marginTop: '8px' }}
                  />
                  <Text type="secondary">
                    <br />
                    Choose between 3 and 50 slides for your course
                  </Text>
                </div>
                <div>
                  <Checkbox
                    checked={generateImages}
                    onChange={(e) => setGenerateImages(e.target.checked)}
                  >
                    <Text strong>Generate AI Images (DALL-E 3)</Text>
                  </Checkbox>
                  <br />
                  <Text type="secondary" style={{ marginLeft: '24px' }}>
                    Generate unique images for each slide using AI (adds processing time and costs)
                  </Text>
                </div>
                <div>
                  <Checkbox
                    checked={generateVoiceover}
                    onChange={(e) => setGenerateVoiceover(e.target.checked)}
                  >
                    <Text strong>Generate Voiceover (Text-to-Speech)</Text>
                  </Checkbox>
                  <br />
                  <Text type="secondary" style={{ marginLeft: '24px' }}>
                    Generate AI voiceover narration for each slide using OpenAI TTS (adds processing time and costs)
                  </Text>
                </div>
                <div>
                  <Checkbox
                    checked={generateQuestions}
                    onChange={(e) => setGenerateQuestions(e.target.checked)}
                  >
                    <Text strong>Generate Live Questions</Text>
                  </Checkbox>
                  <br />
                  <Text type="secondary" style={{ marginLeft: '24px' }}>
                    Generate a multiple choice quiz question for each slide
                  </Text>
                </div>
                <Button
                  type="primary"
                  size="large"
                  icon={<BulbOutlined />}
                  onClick={handleGenerateCourse}
                  loading={genLoading}
                  block
                >
                  {genLoading ? 'Generating Course...' : 'Generate Course with AI'}
                </Button>
                {genLoading && (
                  <div>
                    <Progress percent={50} status="active" />
                    <Text type="secondary">AI is analyzing your document and creating slides...</Text>
                  </div>
                )}
              </Space>
            </Card>
          )}

          {/* Slide Viewer */}
          {slides.length > 0 && (
            <div style={{
              display: 'flex',
              flexDirection: window.innerWidth > 768 ? 'row' : 'column',
              gap: window.innerWidth > 768 ? '24px' : '16px'
            }}>
              <div style={{ flex: chatOpen && window.innerWidth > 768 ? 2 : 1 }}>
                <Card
                  title={
                    <div style={{
                      display: 'flex',
                      justifyContent: 'space-between',
                      alignItems: 'center',
                      flexWrap: 'wrap',
                      gap: '8px'
                    }}>
                      <Space size="small">
                        <FileTextOutlined />
                        <Text strong style={{ fontSize: window.innerWidth > 768 ? '14px' : '12px' }}>
                          Slide {currentSlide + 1} of {slides.length}
                        </Text>
                      </Space>
                      <Space size="small" wrap>
                        {slides[currentSlide]?.audio_url && (
                          <Button
                            type="default"
                            icon={<SoundOutlined />}
                            onClick={() => {
                              // Stop any currently playing audio
                              if (audioRef.current) {
                                audioRef.current.pause()
                                audioRef.current = null
                              }

                              const baseUrl = import.meta.env.DEV ? 'http://localhost:8080' : ''
                              const audio = new Audio(`${baseUrl}${slides[currentSlide].audio_url}`)
                              audioRef.current = audio
                              audio.play()

                              // Auto-advance to next slide when audio finishes
                              audio.onended = () => {
                                if (currentSlide < slides.length - 1) {
                                  setCurrentSlide(currentSlide + 1)
                                }
                                audioRef.current = null
                              }
                            }}
                            size={window.innerWidth > 768 ? 'middle' : 'small'}
                          >
                            {window.innerWidth > 768 ? 'Present' : ''}
                          </Button>
                        )}
                        {slides[currentSlide]?.instructor_script && (
                          <Button
                            icon={<ReadOutlined />}
                            onClick={() => setNotesVisible(true)}
                            size={window.innerWidth > 768 ? 'middle' : 'small'}
                          >
                            {window.innerWidth > 768 ? 'Script' : ''}
                          </Button>
                        )}
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
                          size={window.innerWidth > 768 ? 'middle' : 'small'}
                        >
                          {window.innerWidth > 768 ? 'Previous' : ''}
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
                          size={window.innerWidth > 768 ? 'middle' : 'small'}
                        >
                          {window.innerWidth > 768 ? 'Next' : ''}
                        </Button>
                      </Space>
                    </div>
                  }
                  style={{ minHeight: '500px' }}
                  bodyStyle={{ padding: 0 }}
                >
                  {(() => {
                    const slide = slides[currentSlide]
                    const themeColors = {
                      blue: { bg: '#e6f7ff', border: '#1890ff', text: '#001529' },
                      green: { bg: '#f6ffed', border: '#52c41a', text: '#135200' },
                      purple: { bg: '#f9f0ff', border: '#722ed1', text: '#22075e' },
                      orange: { bg: '#fff7e6', border: '#fa8c16', text: '#ad4e00' },
                      gradient: { bg: 'linear-gradient(135deg, #667eea 0%, #764ba2 100%)', border: '#667eea', text: '#fff' },
                    }
                    const theme = themeColors[slide?.theme as keyof typeof themeColors] || themeColors.blue
                    const isGradient = slide?.theme === 'gradient'

                    return (
                      <div
                        style={{
                          background: theme.bg,
                          minHeight: window.innerWidth > 768 ? '450px' : '300px',
                          padding: window.innerWidth > 768 ? '32px' : '16px',
                          borderLeft: `4px solid ${theme.border}`,
                        }}
                      >
                        {/* Image Section */}
                        {slide?.image_url && (
                          <div style={{ marginBottom: window.innerWidth > 768 ? '24px' : '16px', textAlign: 'center' }}>
                            <img
                              src={slide.image_url}
                              alt={slide.image_prompt || slide.title}
                              style={{
                                maxWidth: '100%',
                                maxHeight: window.innerWidth > 768 ? '300px' : '200px',
                                borderRadius: '8px',
                                boxShadow: '0 4px 12px rgba(0,0,0,0.15)',
                              }}
                            />
                          </div>
                        )}

                        {/* Title based on layout */}
                        {slide?.layout === 'title' ? (
                          <div style={{ textAlign: 'center', padding: '48px 0' }}>
                            <Title level={1} style={{ color: isGradient ? '#fff' : theme.text, marginBottom: '16px' }}>
                              {slide.title}
                            </Title>
                            {slide.content && (
                              <Text style={{ fontSize: '18px', color: isGradient ? '#fff' : theme.text, opacity: 0.9 }}>
                                {slide.content}
                              </Text>
                            )}
                          </div>
                        ) : slide?.layout === 'quote' ? (
                          <div>
                            <Title level={3} style={{ color: isGradient ? '#fff' : theme.text, marginBottom: '24px' }}>
                              {slide.title}
                            </Title>
                            <Card
                              style={{
                                background: isGradient ? 'rgba(255,255,255,0.2)' : '#fff',
                                borderLeft: `4px solid ${theme.border}`,
                                fontStyle: 'italic',
                              }}
                            >
                              <Paragraph
                                style={{
                                  fontSize: '20px',
                                  lineHeight: '1.8',
                                  whiteSpace: 'pre-line',
                                  color: isGradient ? '#fff' : theme.text,
                                  margin: 0,
                                }}
                              >
                                "{slide.content}"
                              </Paragraph>
                            </Card>
                          </div>
                        ) : slide?.layout === 'highlight' ? (
                          <div>
                            <Title level={2} style={{ color: isGradient ? '#fff' : theme.text, marginBottom: '24px' }}>
                              {slide.title}
                            </Title>
                            <Card
                              style={{
                                background: isGradient ? 'rgba(255,255,255,0.2)' : '#fff',
                                border: `2px solid ${theme.border}`,
                              }}
                            >
                              <Paragraph
                                style={{
                                  fontSize: '18px',
                                  lineHeight: '1.8',
                                  whiteSpace: 'pre-line',
                                  color: isGradient ? '#fff' : theme.text,
                                  fontWeight: 500,
                                  margin: 0,
                                }}
                              >
                                {slide.content}
                              </Paragraph>
                            </Card>
                          </div>
                        ) : slide?.layout === 'comparison' ? (
                          <div>
                            <Title level={2} style={{ color: isGradient ? '#fff' : theme.text, marginBottom: '24px' }}>
                              {slide.title}
                            </Title>
                            <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '16px' }}>
                              {slide.content.split('\n\n').map((section, idx) => (
                                <Card
                                  key={idx}
                                  style={{
                                    background: isGradient ? 'rgba(255,255,255,0.2)' : '#fff',
                                    height: '100%',
                                  }}
                                >
                                  <Paragraph
                                    style={{
                                      fontSize: '16px',
                                      lineHeight: '1.8',
                                      whiteSpace: 'pre-line',
                                      color: isGradient ? '#fff' : theme.text,
                                      margin: 0,
                                    }}
                                  >
                                    {section}
                                  </Paragraph>
                                </Card>
                              ))}
                            </div>
                          </div>
                        ) : (
                          // Default layout
                          <div>
                            <Title level={2} style={{ color: isGradient ? '#fff' : theme.text, marginBottom: '16px' }}>
                              {slide.title}
                            </Title>
                            <Divider style={{ borderColor: theme.border, opacity: 0.5 }} />
                            <Paragraph
                              style={{
                                fontSize: '16px',
                                lineHeight: '1.8',
                                whiteSpace: 'pre-line',
                                color: isGradient ? '#fff' : theme.text,
                              }}
                            >
                              {slide.content}
                            </Paragraph>
                          </div>
                        )}
                      </div>
                    )
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
                    style={{ marginTop: '24px' }}
                  >
                    <div>
                      <Paragraph style={{ fontSize: '16px', fontWeight: 500, marginBottom: '16px' }}>
                        {questions[currentSlide].question}
                      </Paragraph>
                      <Space direction="vertical" style={{ width: '100%' }}>
                        {questions[currentSlide].options.map((option, idx) => {
                          const slideAnswer = quizAnswers.find(a => a.slideId === slides[currentSlide].id)
                          const isSelected = slideAnswer?.selectedAnswer === idx
                          const isCorrect = questions[currentSlide].correct_answer === idx
                          const showResult = slideAnswer !== undefined

                          let buttonType: 'default' | 'primary' | 'dashed' = 'default'
                          let buttonStyle: React.CSSProperties = {
                            width: '100%',
                            textAlign: 'left',
                            height: 'auto',
                            padding: window.innerWidth > 768 ? '12px' : '16px',
                            fontSize: window.innerWidth > 768 ? '14px' : '16px'
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
                              type={buttonType}
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
                    </div>
                  </Card>
                )}
              </div>

              {/* Chat Panel */}
              {chatOpen && (
                <div style={{ flex: 1, minWidth: window.innerWidth > 768 ? '400px' : '100%' }}>
                  <Card
                    title={
                      <div style={{
                        display: 'flex',
                        justifyContent: 'space-between',
                        alignItems: 'center',
                        flexWrap: 'wrap',
                        gap: '8px'
                      }}>
                        <Space size="small">
                          <RobotOutlined />
                          <Text strong style={{ fontSize: window.innerWidth > 768 ? '14px' : '12px' }}>
                            AI Assistant
                          </Text>
                          <Badge count={messages.length} size="small" />
                        </Space>
                        {questions.length > 0 && (
                          <Space size="small">
                            <Text strong style={{ fontSize: window.innerWidth > 768 ? '14px' : '12px' }}>
                              Score:
                            </Text>
                            <Tag color="blue" style={{ fontSize: window.innerWidth > 768 ? '14px' : '12px' }}>
                              {quizAnswers.filter(a => a.isCorrect).length} / {questions.length}
                            </Tag>
                          </Space>
                        )}
                      </div>
                    }
                    style={{
                      height: window.innerWidth > 768 ? '600px' : 'auto',
                      minHeight: window.innerWidth > 768 ? '600px' : '400px',
                      display: 'flex',
                      flexDirection: 'column'
                    }}
                    bodyStyle={{ flex: 1, display: 'flex', flexDirection: 'column', padding: 0 }}
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
                                    <Text strong>{msg.role === 'user' ? 'You' : 'AI Assistant'}</Text>
                                    <Paragraph style={{ marginTop: '8px', marginBottom: 0 }}>
                                      {msg.content}
                                    </Paragraph>
                                    {msg.citations && msg.citations.length > 0 && (
                                      <div style={{ marginTop: '8px' }}>
                                        <Text type="secondary" style={{ fontSize: '12px' }}>
                                          Sources:{' '}
                                        </Text>
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
                          placeholder="Ask a question about the course..."
                          value={question}
                          onChange={(e) => setQuestion(e.target.value)}
                          onPressEnter={handleAsk}
                          disabled={loading}
                          size="large"
                        />
                        <Button
                          type="primary"
                          icon={<SendOutlined />}
                          onClick={handleAsk}
                          loading={loading}
                          disabled={!question.trim()}
                          size="large"
                        >
                          Send
                        </Button>
                      </Space.Compact>
                    </div>
                  </Card>
                </div>
              )}
            </div>
          )}
        </Content>
      </Layout>

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
    </Layout>
  )
}

export default App
