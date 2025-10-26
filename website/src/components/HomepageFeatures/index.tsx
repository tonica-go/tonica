import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  emoji: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Proto-First Architecture',
    emoji: '📝',
    description: (
      <>
        Define your APIs with Protocol Buffers and get automatic gRPC, REST, and OpenAPI documentation.
        Type-safe, contract-first development out of the box.
      </>
    ),
  },
  {
    title: 'Multiple Run Modes',
    emoji: '🎯',
    description: (
      <>
        Run as All-in-One for development, or split into Service, Worker, Consumer, and Gateway modes for production.
        Scale components independently based on your needs.
      </>
    ),
  },
  {
    title: 'Built-in Observability',
    emoji: '📊',
    description: (
      <>
        OpenTelemetry tracing, Prometheus metrics, and structured logging (slog) included.
        Production-ready monitoring and debugging from day one.
      </>
    ),
  },
  {
    title: 'Graceful Shutdown',
    emoji: '⚡',
    description: (
      <>
        Proper lifecycle management for HTTP/gRPC servers, workers, and consumers.
        Zero-downtime deployments with clean shutdown coordination.
      </>
    ),
  },
  {
    title: 'Developer Experience',
    emoji: '🚀',
    description: (
      <>
        CLI tools for scaffolding, fluent API for custom routes, and comprehensive testing support.
        Start building in minutes, not hours.
      </>
    ),
  },
  {
    title: 'Production Ready',
    emoji: '🏗️',
    description: (
      <>
        Database support (PostgreSQL, MySQL, SQLite), Redis caching, Temporal workers, and message consumers.
        Everything you need for real-world microservices.
      </>
    ),
  },
];

function Feature({title, emoji, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4')}>
      <div className="text--center">
        <div style={{fontSize: '4rem', marginBottom: '1rem'}}>{emoji}</div>
      </div>
      <div className="text--center padding-horiz--md">
        <Heading as="h3">{title}</Heading>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
